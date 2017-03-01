/*
   Copyright 2016-2017 Red Hat, Inc. and/or its affiliates
   and other contributors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package manager

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/collector/tracker"
	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/config/tags"
	agentmetrics "github.com/hawkular/hawkular-openshift-agent/emitter/metrics"
	"github.com/hawkular/hawkular-openshift-agent/emitter/status"
	"github.com/hawkular/hawkular-openshift-agent/log"
	"github.com/hawkular/hawkular-openshift-agent/util/expand"
	"github.com/hawkular/hawkular-openshift-agent/util/stopwatch"
)

// MetricsCollectorManager is responsible for periodically collecting metrics from many different endpoints.
type MetricsCollectorManager struct {
	TickersLock    *sync.Mutex
	Tickers        map[collector.CollectorID]*time.Ticker
	Config         *config.Config
	metricsTracker tracker.MetricsTracker
	metricsChan    chan []hmetrics.MetricHeader
	metricDefsChan chan []hmetrics.MetricDefinition
}

func NewMetricsCollectorManager(conf *config.Config, metricsChan chan []hmetrics.MetricHeader, metricDefsChan chan []hmetrics.MetricDefinition) *MetricsCollectorManager {
	mcm := &MetricsCollectorManager{
		TickersLock:    &sync.Mutex{},
		Tickers:        make(map[collector.CollectorID]*time.Ticker),
		Config:         conf,
		metricsTracker: tracker.NewMetricsTracker(conf.Kubernetes.Max_Metrics_Per_Pod),
		metricsChan:    metricsChan,
		metricDefsChan: metricDefsChan,
	}

	// allow the status reporter to know how many metrics we are collecting
	status.StatusReport.SetMetricsTracker(&mcm.metricsTracker)

	log.Tracef("New metrics collector manager has been created. config=%v", conf)
	log.Infof("Pods will be limited to a maximum of [%v] metrics", conf.Kubernetes.Max_Metrics_Per_Pod)
	log.Infof("Endpoints will be limited to a minimum collection interval of [%v]", conf.Collector.Minimum_Collection_Interval)

	return mcm
}

func (mcm *MetricsCollectorManager) StartCollectingEndpoints(endpoints []collector.Endpoint) {
	if endpoints != nil {
		for _, e := range endpoints {
			id := collector.CollectorID{
				PodID:      "",
				EndpointID: fmt.Sprintf("X|X|%v|%v", e.Type, e.URL),
			}
			if c, err := CreateMetricsCollector(id, mcm.Config.Identity, e, nil); err != nil {
				m := fmt.Sprintf("Will not start collecting for endpoint [%v]. err=%v", id.EndpointID, err)
				log.Warning(m)
				mcm.NotCollecting(id, m)
			} else {
				mcm.StartCollecting(c)
			}
		}
	}
	return

}

// StartCollecting will collect metrics every "collection interval" seconds in a go routine.
// If a metrics collector with the same ID is already collecting metrics, it will be stopped
// and the given new collector will take its place.
func (mcm *MetricsCollectorManager) StartCollecting(theCollector collector.MetricsCollector) {

	id := theCollector.GetID()

	// if there was an old ticker still running for this collector, stop it
	mcm.StopCollecting(id)

	// ensures we track this pod/endpoint even on error
	mcm.metricsTracker.AddMetricsFromCollector(id, nil)

	if theCollector.GetEndpoint().IsEnabled() == false {
		m := fmt.Sprintf("Will not collect metrics from [%v] - it has been disabled.", id)
		log.Info(m)
		status.StatusReport.SetEndpointMessage(id, m)
		return
	}

	// determine the collection interval
	var collectionInterval, minimumInterval time.Duration
	var parseErr error
	collectionIntervalString := theCollector.GetEndpoint().Collection_Interval
	if collectionIntervalString == "" {
		log.Debugf("Collection interval for [%v] is not defined - setting to the default interval [%v]",
			id, mcm.Config.Collector.Default_Collection_Interval)
		collectionIntervalString = mcm.Config.Collector.Default_Collection_Interval
	}
	if collectionInterval, parseErr = time.ParseDuration(collectionIntervalString); parseErr != nil {
		log.Warningf("Collection interval for [%v] is invalid - setting to the default interval [%v]. err=%v",
			id, mcm.Config.Collector.Default_Collection_Interval, parseErr)
		if collectionInterval, parseErr = time.ParseDuration(mcm.Config.Collector.Default_Collection_Interval); parseErr != nil {
			log.Warningf("Default collection interval is invalid. This is a bug, please report. err=%v", parseErr)
			collectionInterval = time.Minute * 5
		}
	}
	if minimumInterval, parseErr = time.ParseDuration(mcm.Config.Collector.Minimum_Collection_Interval); parseErr == nil {
		if collectionInterval < minimumInterval {
			log.Warningf("Collection interval for [%v] is [%v] which is lower than the minimum allowed [%v]. Setting it to the minimum allowed.",
				id, collectionInterval.String(), minimumInterval.String())
			collectionInterval = minimumInterval
		}
	} else {
		log.Warningf("Minimum collection interval is invalid. This is a bug, please report. err=%v", parseErr)
	}

	// log some information about the new collector
	log.Infof("START collecting metrics from [%v] every [%v]", id, collectionInterval)
	status.StatusReport.AddLogMessage(fmt.Sprintf("START collection: %v (interval=%v)", id, collectionInterval))
	status.StatusReport.SetEndpointMessage(id, "STARTING")

	// lock access to the Tickers array
	mcm.TickersLock.Lock()
	defer mcm.TickersLock.Unlock()

	// now periodically collect the metrics within a go routine
	ticker := time.NewTicker(collectionInterval)
	mcm.Tickers[id] = ticker
	go func() {
		// we need these to expand tokens in the IDs
		mappingFunc := expand.MappingFunc(expand.MappingFuncConfig{
			Env:                   theCollector.GetAdditionalEnvironment(),
			UseOSEnv:              false,
			DoNotExpandIfNotFound: true,
		})
		mappingFuncWithOsEnv := expand.MappingFunc(expand.MappingFuncConfig{
			Env:      theCollector.GetAdditionalEnvironment(),
			UseOSEnv: true,
		})

		// metric names may have ${x} tokens in them - we need to expand them now before we do anything with metrics
		for i, mm := range theCollector.GetEndpoint().Metrics {
			theCollector.GetEndpoint().Metrics[i].Name = os.Expand(mm.Name, mappingFunc)
		}

		// cache that tracks what metric definitions we already created - key is full metric ID (prefixed and expanded)
		metricDefinitionsDeclared := make(map[string]collector.MonitoredMetric, 0)

		// Cache the endpoint metrics to be collected in a map keyed on name for quick lookups.
		// This cache will be empty if the endpoint was told to collect all metrics.
		monitoredMetricsByNameMap := make(map[string]collector.MonitoredMetric, len(theCollector.GetEndpoint().Metrics))
		for _, mm := range theCollector.GetEndpoint().Metrics {
			monitoredMetricsByNameMap[mm.Name] = mm
		}

		// for each collection interval, perform endpoint metric collection (this also creates/updates metric definitions as needed)
		for _ = range ticker.C {
			timer := stopwatch.NewStopwatch()
			collectedMetrics, err := theCollector.CollectMetrics()
			timer.MarkTime()

			if err != nil {
				msg := fmt.Sprintf("Failed to collect metrics from [%v] at [%v]. err=%v", id, timer.StartTime().Format(time.RFC1123Z), err)
				log.Warning(msg)
				status.StatusReport.SetEndpointMessage(id, msg)
			} else {
				// if any metric definitions need to be created, they will be noted here - key is full and expanded metric ID
				metricDefinitionsNeeded := make(map[string]collector.MonitoredMetric, 0)

				// if metrics are labeled and thus need to be split, they will be stored in here
				var allSplitMetrics []hmetrics.MetricHeader

				for i, collectedMetric := range collectedMetrics {
					// If the endpoint has a list of metrics, make sure we only collect what we were told to collect.
					// If the endpoint has no metrics listed, it means we are to collect all of them.
					// Remember the collected metric's ID is really the metric name.
					var monitoredMetric collector.MonitoredMetric
					if len(monitoredMetricsByNameMap) > 0 {
						var ok bool
						monitoredMetric, ok = monitoredMetricsByNameMap[collectedMetric.ID] // remember, the collector returned metric IDs which are our metric names
						if !ok {
							collectedMetrics[i].ID = "" // unknown metric that will need to be removed
							log.Warningf("Metric [%v] was collected but wasn't expected from endpoint [%v]", collectedMetric.ID, id)
							continue
						}
					} else {
						// endpoint wasn't given any monitoredMetric data so just create one based on the collected metric data.
						monitoredMetric = collector.MonitoredMetric{
							ID:   collectedMetric.ID,
							Name: collectedMetric.ID,
							Type: collectedMetric.Type,
						}
					}

					// we want to prefix the metric ID and replace the ${x} tokens but leave unmapped ${x} untouched so we know if we need to split the metric
					collectedMetrics[i].ID = os.Expand(mcm.Config.Collector.Metric_ID_Prefix, mappingFuncWithOsEnv) + os.Expand(monitoredMetric.ID, mappingFunc)

					// To support endpoints that report different time series based on labels (e.g. Prometheus)
					// metric IDs can be declared with ${label} tokens. This means metrics with the same name
					// but have labels can really represent different metric IDs. We need to "split" these metrics
					// up and make sure we create metric definitions for each of these metric IDs.
					// For example, an endpoint can declare a metric whose name is "request_time" with these datapoints collected:
					//   request_time{method="GET"} (has one label where key=method and value=GET)
					//   request_time{method="POST"} (has one label where key=method and value=POST)
					// They have the same metric name "request_time" but the endpoint can declare this metric with an ID of
					// "request_time_${method}". In that case, these datapoints result in 2 metric definitions with 2 IDs.
					// Note that if a metric has an ID that does not define ${label} tokens explicitly, but that metric
					// has data points with tags, then those metrics will be split up by default. The ID will
					// have a default format with the sorted list of tags appended to the end.

					if !strings.Contains(collectedMetrics[i].ID, "${") {
						// look at each data point and extract each label name. Use a map to avoid duplicates.
						keysMap := make(map[string]bool, 0)
						for _, datapt := range collectedMetric.Data {
							for k, _ := range datapt.Tags {
								keysMap[k] = true
							}
						}
						if len(keysMap) > 0 {
							// put the keys in an array and sort the array - we want the tags to be in order
							keys := make([]string, len(keysMap))
							keyIndex := 0
							for k, _ := range keysMap {
								keys[keyIndex] = k
								keyIndex++
							}
							sort.Strings(keys)
							var keysString bytes.Buffer
							comma := ""
							for _, k := range keys {
								keysString.WriteString(fmt.Sprintf("%v%v=${%v}", comma, k, k))
								comma = ","
							}
							collectedMetrics[i].ID = fmt.Sprintf("%v{%v}", collectedMetrics[i].ID, keysString.String())
							log.Tracef("Metric [%v] to be split into separate metrics using ID [%v] for endpoint [%v]", monitoredMetric.Name, collectedMetrics[i].ID, id)
						}
					}

					if strings.Contains(collectedMetrics[i].ID, "${") {
						splitMetrics := make([]hmetrics.MetricHeader, len(collectedMetric.Data))
						for j, datapt := range collectedMetric.Data {
							datapointTags := datapt.Tags
							// comment the below line if we want to tag the datapoints, too
							datapt.Tags = map[string]string{}
							splitMetrics[j] = hmetrics.MetricHeader{
								Tenant: collectedMetric.Tenant,
								Type:   collectedMetric.Type,
								Data:   []hmetrics.Datapoint{datapt},
								ID: os.Expand(collectedMetrics[i].ID, expand.MappingFunc(expand.MappingFuncConfig{
									UseOSEnv: false,
									Env:      datapointTags,
								})),
							}

							// if we need to create the metric definition, remember it
							if _, ok := metricDefinitionsDeclared[splitMetrics[j].ID]; !ok {
								// monitoredMetric needs to have the data tags because they will be used as tags on the metric def
								monitoredMetricCopy := monitoredMetric.Clone()
								if len(datapointTags) > 0 {
									if monitoredMetricCopy.Tags == nil {
										monitoredMetricCopy.Tags = tags.Tags{}
									}
									for k, v := range datapointTags {
										monitoredMetricCopy.Tags["@@@"+k] = v // @@@ denotes it is a tag from the datapoint
									}
								}
								metricDefinitionsNeeded[splitMetrics[j].ID] = monitoredMetricCopy
							}
						}

						log.Tracef("Split metric [%v] into [%v] separate metrics for endpoint [%v]", collectedMetrics[i].ID, len(splitMetrics), id)
						collectedMetrics[i].ID = ""                                // this is a metric that will need to be removed - only the split-out metrics are needed
						allSplitMetrics = append(allSplitMetrics, splitMetrics...) // remember these split metrics
					} else {
						// if we need to create the metric definition, remember it
						if _, ok := metricDefinitionsDeclared[collectedMetrics[i].ID]; !ok {
							metricDefinitionsNeeded[collectedMetrics[i].ID] = monitoredMetric
						}
					}
				}

				// we may have unknown or split-up metrics - remove them
				i := 0
				for _, collectedMetric := range collectedMetrics {
					if collectedMetric.ID != "" {
						collectedMetrics[i] = collectedMetric
						i++
					}
				}
				collectedMetrics = collectedMetrics[:i]

				// combine the known metrics and all the split metrics - this ends up with all the metrics we want to store
				collectedMetrics = append(collectedMetrics, allSplitMetrics...)

				// Make sure we only store the maximum allowed.
				// If we go over, we must remove the metrics we just collected so we do not store them.
				overflow := mcm.metricsTracker.AddMetricsFromCollector(id, collectedMetrics)
				if len(overflow) > 0 {
					log.Warningf("Reached max limit of metrics for [%v] - discarding [%v] collected metrics", id, len(overflow))
					status.StatusReport.SetEndpointMessage(id,
						fmt.Sprintf("METRIC LIMIT EXCEEDED. Last collection at [%v] gathered [%v] metrics, [%v] were discarded, in [%v]",
							timer.StartTime().Format(time.RFC1123Z), len(collectedMetrics), len(overflow), timer))

					i := 0
					for _, collectedMetric := range collectedMetrics {
						keep := true
						for _, doomedMetric := range overflow {
							if doomedMetric.ID == collectedMetric.ID {
								keep = false
								break
							}
						}
						if keep {
							collectedMetrics[i] = collectedMetric
							i++
						}
					}
					collectedMetrics = collectedMetrics[:i]

					// Do not create metric definitions for the overflowed metrics.
					// While we are looping over the discarded metrics, let's log them.
					for _, doomedMetric := range overflow {
						delete(metricDefinitionsNeeded, doomedMetric.ID)
						log.Tracef("Discarding metric [%v] from [%v]", doomedMetric.ID, id)
					}
				} else {
					status.StatusReport.SetEndpointMessage(id,
						fmt.Sprintf("OK. Last collection at [%v] gathered [%v] metrics in [%v]",
							timer.StartTime().Format(time.RFC1123Z), len(collectedMetrics), timer))
				}

				// now send all metrics for storage
				mcm.metricsChan <- collectedMetrics

				// create the missing metric definitions
				if len(metricDefinitionsNeeded) > 0 {
					log.Tracef("Need to create/update [%v] metric definitions for endpoint [%v]", len(metricDefinitionsNeeded), id)
					if err := mcm.createMetricDefinition(theCollector, metricDefinitionsNeeded); err == nil {
						for k, v := range metricDefinitionsNeeded {
							metricDefinitionsDeclared[k] = v
						}
					}
				}

				// record keeping to update the agent's own metrics
				pc, ec, mc := mcm.metricsTracker.GetCounts()
				agentmetrics.Metrics.DataPointsCollected.Add(float64(len(collectedMetrics)))
				agentmetrics.Metrics.MonitoredPods.Set(float64(pc))
				agentmetrics.Metrics.MonitoredEndpoints.Set(float64(ec))
				agentmetrics.Metrics.MonitoredMetrics.Set(float64(mc))
			}
		}
	}()
}

// StopCollecting will stop metric collection for the collector that has the given ID.
func (mcm *MetricsCollectorManager) StopCollecting(collectorId collector.CollectorID) {
	// lock access to the Tickers array
	mcm.TickersLock.Lock()
	defer mcm.TickersLock.Unlock()

	ticker, ok := mcm.Tickers[collectorId]
	if ok {
		log.Infof("STOP collecting metrics from [%v]", collectorId)
		status.StatusReport.AddLogMessage(fmt.Sprintf("STOP collection: %v", collectorId))
		ticker.Stop()
		delete(mcm.Tickers, collectorId)
	}

	// ensure we take it out of the status report, even if no ticker was running on it
	status.StatusReport.SetEndpointMessage(collectorId, "")

	mcm.metricsTracker.PurgeMetricsForCollectorEndpoint(collectorId)
	pc, ec, mc := mcm.metricsTracker.GetCounts()
	agentmetrics.Metrics.MonitoredPods.Set(float64(pc))
	agentmetrics.Metrics.MonitoredEndpoints.Set(float64(ec))
	agentmetrics.Metrics.MonitoredMetrics.Set(float64(mc))

}

// StopCollectingAll halts all metric collections.
func (mcm *MetricsCollectorManager) StopCollectingAll() {
	// lock access to the Tickers array
	mcm.TickersLock.Lock()
	defer mcm.TickersLock.Unlock()

	log.Infof("STOP collecting all metrics from all endpoints")
	status.StatusReport.AddLogMessage("STOP collecting all metrics from all endpoints")
	for id, ticker := range mcm.Tickers {
		ticker.Stop()
		delete(mcm.Tickers, id)
	}

	// purge status report, metric tracker, metrics
	status.StatusReport.DeleteAllEndpointMessages()
	mcm.metricsTracker.PurgeAllMetrics()
	agentmetrics.Metrics.MonitoredPods.Set(float64(0))
	agentmetrics.Metrics.MonitoredEndpoints.Set(float64(0))
	agentmetrics.Metrics.MonitoredMetrics.Set(float64(0))
}

// NotCollecting is a way to notify the collection manager that there is an endpoint
// that exists but is one we are not collecting for whatever reason. The collection
// manager can use this information to update the status report and the metrics tracker.
func (mcm *MetricsCollectorManager) NotCollecting(id collector.CollectorID, reason string) {
	mcm.metricsTracker.AddMetricsFromCollector(id, nil) // so we track this pod/endpoint
	status.StatusReport.SetEndpointMessage(id, reason)
}

func (mcm *MetricsCollectorManager) createMetricDefinition(theCollector collector.MetricsCollector,
	metricDefsNeeded map[string]collector.MonitoredMetric) (err error) {

	// short circuit if there is nothing to do
	if len(metricDefsNeeded) == 0 {
		return nil
	}

	endpoint := theCollector.GetEndpoint()
	additionalEnv := theCollector.GetAdditionalEnvironment()

	// get the metric names we need details for. (eliminate possible duplicates)
	metricNamesSet := make(map[string]bool, len(metricDefsNeeded))
	for _, v := range metricDefsNeeded {
		metricNamesSet[v.Name] = true
	}
	metricNamesArrIndex := 0
	metricNamesArr := make([]string, len(metricNamesSet))
	for n, _ := range metricNamesSet {
		metricNamesArr[metricNamesArrIndex] = n
		metricNamesArrIndex++
	}

	// ask the collector for all details on all the named metrics we need
	log.Tracef("Collecting [%v] metric details for endpoint [%v]", len(metricNamesArr), endpoint)
	metricDetails, e := theCollector.CollectMetricDetails(metricNamesArr)
	if e != nil {
		metricDetails = make([]collector.MetricDetails, 0)
		msg := fmt.Sprintf("Failed to obtain metric details - metric definitions may be incomplete. err=%v", err)
		log.Warning(msg)
		status.StatusReport.SetEndpointMessage(theCollector.GetID(), msg)
		// Keep going to create the defs, but we'll return this error so we'll try again later to update
		// the defs with the full details when we can get them.
		err = e
	}

	metricDefs := make([]hmetrics.MetricDefinition, len(metricDefsNeeded))
	i := 0

	for metricId, monitoredMetric := range metricDefsNeeded {

		var metricDetail collector.MetricDetails

		// find the metric details for the metric we are currently working on
		for _, m := range metricDetails {
			if monitoredMetric.Name == m.Name {
				metricDetail = m
			}
		}

		// Some of the tags in the given monitoredMetric were those given to us directly by the endpoint in the datapoint.
		// We need to extract them out and use only those when building the default description.
		datapointTags := tags.Tags{}
		for k, v := range monitoredMetric.Tags {
			if strings.HasPrefix(k, "@@@") {
				actualTagName := strings.TrimPrefix(k, "@@@")
				datapointTags[actualTagName] = v
				delete(monitoredMetric.Tags, k)
				monitoredMetric.Tags[actualTagName] = v // put it back with the actual name so it goes on our metric def later
			}
		}

		// NOTE: If the metric type was declared, we use it. Otherwise, we look at
		// metric details to see if there is a type available and if so, use it.
		// This is to support the fact that Prometheus indicates the type in the metric endpoint
		// so there is no need to ask the user to define it in a configuration file.
		// The same is true with metric description as well.
		metricType := monitoredMetric.Type
		metricDescription := monitoredMetric.Description
		if metricType == "" {
			metricType = metricDetail.MetricType
			if metricType == "" {
				metricType = hmetrics.Gauge
				log.Warningf("Metric definition [%v] type cannot be determined for endpoint [%v]. Will assume its type is [%v] ", monitoredMetric.Name, endpoint, metricType)
			}
		}
		if metricDescription == "" {
			var descriptionBuffer bytes.Buffer
			if metricDetail.Description != "" {
				descriptionBuffer.WriteString(metricDetail.Description)
			} else {
				descriptionBuffer.WriteString(monitoredMetric.Name)
			}

			// If the metric datapoint had tags on it, let the default description include the tags to make the description unique.
			// Put the tag names in an array and sort the array - we want the names to be in order.
			if len(datapointTags) > 0 {
				keys := make([]string, len(datapointTags))
				keyIndex := 0
				for k, _ := range datapointTags {
					keys[keyIndex] = k
					keyIndex++
				}
				sort.Strings(keys)
				separator := " {"
				for _, k := range keys {
					descriptionBuffer.WriteString(fmt.Sprintf("%v%v=%v", separator, k, monitoredMetric.Tags[k]))
					separator = ","
				}
				descriptionBuffer.WriteString("}")
			}

			metricDescription = descriptionBuffer.String()
		}

		// Now add the fixed tag of "units".
		units, err := collector.GetMetricUnits(monitoredMetric.Units)
		if err != nil {
			log.Warningf("Units for metric definition [%v] for endpoint [%v] is invalid. Assigning unit value to [%v]. err=%v", monitoredMetric.Name, endpoint, units.Symbol, err)
		}

		// Define additional envvars with pod specific data for use in replacing ${env} tokens in tags.
		env := map[string]string{
			"METRIC:name":        monitoredMetric.Name,
			"METRIC:id":          metricId,
			"METRIC:units":       units.Symbol,
			"METRIC:description": metricDescription,
		}

		for k, v := range monitoredMetric.Tags {
			env[fmt.Sprintf("METRIC:tag[%v]", k)] = v
		}

		for key, value := range additionalEnv {
			env[key] = value
		}

		// Notice: global tags override metric tags which override endpoint tags.
		// Do NOT allow pods to use agent environment variables since agent env vars may contain
		// sensitive data (such as passwords). Only the global agent config can define tags
		// with env var tokens.
		noOsEnv := expand.MappingFuncConfig{Env: env, UseOSEnv: false}
		withOsEnv := expand.MappingFuncConfig{Env: env, UseOSEnv: true}
		globalTags := mcm.Config.Collector.Tags.ExpandTokens(withOsEnv)
		endpointTags := endpoint.Tags.ExpandTokens(noOsEnv)

		// The metric tags will consist of the custom tags as well as the fixed tags.
		// First start with the custom tags...
		metricTags := monitoredMetric.Tags.ExpandTokens(noOsEnv)

		// Now add the fixed tag of "description". This is optional.
		if metricDescription != "" {
			metricTags["description"] = metricDescription
		}

		// Now add the fixed tag of "units". This is optional.
		if units.Symbol != "" {
			metricTags["units"] = units.Symbol
		}

		// put all the tags together for the full list of tags to be applied to this metric definition
		allMetricTags := tags.Tags{}
		allMetricTags.AppendTags(endpointTags) // endpoint tags are overridden by
		allMetricTags.AppendTags(metricTags)   // metric tags which are overriden by
		allMetricTags.AppendTags(globalTags)   // global tags

		// we can now create the metric definition object
		metricDefs[i] = hmetrics.MetricDefinition{
			Tenant: endpoint.Tenant,
			Type:   metricType,
			ID:     metricId,
			Tags:   map[string]string(allMetricTags),
		}
		i++
	}

	log.Tracef("[%v] metric definitions being declared for endpoint [%v]", len(metricDefs), endpoint)
	mcm.metricDefsChan <- metricDefs
	return
}
