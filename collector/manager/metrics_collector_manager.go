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
	"fmt"
	"os"
	"sync"
	"time"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
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
	Tickers        map[string]*time.Ticker
	Config         *config.Config
	metricsChan    chan []hmetrics.MetricHeader
	metricDefsChan chan []hmetrics.MetricDefinition
}

func NewMetricsCollectorManager(conf *config.Config, metricsChan chan []hmetrics.MetricHeader, metricDefsChan chan []hmetrics.MetricDefinition) *MetricsCollectorManager {
	mcm := &MetricsCollectorManager{
		TickersLock:    &sync.Mutex{},
		Tickers:        make(map[string]*time.Ticker),
		Config:         conf,
		metricsChan:    metricsChan,
		metricDefsChan: metricDefsChan,
	}
	log.Tracef("New metrics collector manager has been created. config=%v", conf)
	return mcm
}

func (mcm *MetricsCollectorManager) StartCollectingEndpoints(endpoints []collector.Endpoint) {
	if endpoints != nil {
		for _, e := range endpoints {
			id := fmt.Sprintf("%v|%v", e.Type, e.URL)
			if c, err := CreateMetricsCollector(id, mcm.Config.Identity, e, nil); err != nil {
				m := fmt.Sprintf("Will not start collecting for endpoint [%v]. err=%v", id, err)
				log.Warning(m)
				status.StatusReport.SetEndpoint(id, m)
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

	id := theCollector.GetId()

	if theCollector.GetEndpoint().IsEnabled() == false {
		m := fmt.Sprintf("Will not collect metrics from [%v] - it has been disabled.", id)
		log.Info(m)
		status.StatusReport.SetEndpoint(id, m)
		return
	}

	// if there was an old ticker still running for this collector, stop it
	mcm.StopCollecting(id)

	// lock access to the Tickers array
	mcm.TickersLock.Lock()
	defer mcm.TickersLock.Unlock()

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
	status.StatusReport.AddLogMessage(fmt.Sprintf("START collection: %v", id))
	status.StatusReport.SetEndpoint(id, "STARTING")

	// now periodically collect the metrics
	ticker := time.NewTicker(collectionInterval)
	mcm.Tickers[id] = ticker
	go func() {

		// we need these to expand tokens in the IDs
		mappingFunc := expand.MappingFunc(false, theCollector.GetAdditionalEnvironment())
		mappingFuncWithEnv := expand.MappingFunc(true, theCollector.GetAdditionalEnvironment())

		// declare the metric definitions - creating new ones and updating existing ones
		metricDetails, err := theCollector.CollectMetricDetails()
		if err != nil {
			metricDetails = make([]collector.MetricDetails, 0)
			msg := fmt.Sprintf("Failed to obtain metric details - metric definitions may be incomplete. err=%v", err)
			log.Warning(msg)
			status.StatusReport.SetEndpoint(id, msg)
		}
		mcm.declareMetricDefinitions(metricDetails, theCollector.GetEndpoint(), theCollector.GetAdditionalEnvironment())

		// now periodically collect the metric data
		for _ = range ticker.C {
			timer := stopwatch.NewStopwatch()
			metrics, err := theCollector.CollectMetrics()
			timer.MarkTime()
			if err != nil {
				msg := fmt.Sprintf("Failed to collect metrics from [%v] at [%v]. err=%v", id, time.Now().Format(time.RFC1123Z), err)
				log.Warning(msg)
				status.StatusReport.SetEndpoint(id, msg)
			} else {
				for i, m := range metrics {
					metrics[i].ID = os.Expand(mcm.Config.Collector.Metric_ID_Prefix, mappingFuncWithEnv) + os.Expand(m.ID, mappingFunc)
				}
				mcm.metricsChan <- metrics

				agentmetrics.Metrics.DataPointsCollected.Add(float64(len(metrics)))
				status.StatusReport.SetEndpoint(id, fmt.Sprintf("OK. Last collection at [%v] gathered [%v] metrics in [%v]", time.Now().Format(time.RFC1123Z), len(metrics), timer))
			}
		}
	}()
}

// StopCollecting will stop metric collection for the collector that has the given ID.
func (mcm *MetricsCollectorManager) StopCollecting(collectorId string) {
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
	status.StatusReport.SetEndpoint(collectorId, "")
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

	// ensure we take them all out of the status report, even for those which there are no tickers
	status.StatusReport.DeleteAllEndpoints()
}

func (mcm *MetricsCollectorManager) declareMetricDefinitions(metricDetails []collector.MetricDetails, endpoint *collector.Endpoint, additionalEnv map[string]string) {

	metricDefs := make([]hmetrics.MetricDefinition, len(endpoint.Metrics))

	for i, metric := range endpoint.Metrics {

		// NOTE: If the metric type was declared, we use it. Otherwise, we look at
		// metric details to see if there is a type available and if so, use it.
		// This is to support the fact that Prometheus indicates the type in the metric endpoint
		// so there is no need to ask the user to define it in a configuration file.
		// The same is true with metric description as well.
		metricType := metric.Type
		metricDescription := metric.Description
		for _, metricDetail := range metricDetails {
			if metricDetail.ID == metric.ID {
				if metricType == "" {
					metricType = metricDetail.MetricType
				}
				if metricDescription == "" {
					metricDescription = metricDetail.Description
				}
				break
			}
		}
		if metricType == "" {
			metricType = hmetrics.Gauge
			log.Warningf("Metric definition [%v] type cannot be determined for endpoint [%v]. Will assume its type is [%v] ", metric.ID, endpoint.String(), metricType)
		}

		// Now add the fixed tag of "units".
		units, err := collector.GetMetricUnits(metric.Units)
		if err != nil {
			log.Warningf("Units for metric definition [%v] for endpoint [%v] is invalid. Assigning unit value to [%v]. err=%v", metric.ID, endpoint.String(), units.Symbol, err)
		}

		// Define additional envvars with pod specific data for use in replacing ${env} tokens in tags.
		env := map[string]string{
			"METRIC:name":        metric.Name,
			"METRIC:id":          metric.ID,
			"METRIC:units":       units.Symbol,
			"METRIC:description": metricDescription,
		}

		for key, value := range additionalEnv {
			env[key] = value
		}

		// For each metric in the endpoint, create a metric def for it.
		// Notice: global tags override metric tags which override endpoint tags.
		// Do NOT allow pods to use agent environment variables since agent env vars may contain
		// sensitive data (such as passwords). Only the global agent config can define tags
		// with env var tokens.
		globalTags := mcm.Config.Collector.Tags.ExpandTokens(true, env)
		endpointTags := endpoint.Tags.ExpandTokens(false, env)

		// we need these to expand tokens in the IDs
		mappingFunc := expand.MappingFunc(false, env)
		mappingFuncWithEnv := expand.MappingFunc(true, env)

		// The metric tags will consist of the custom tags as well as the fixed tags.
		// First start with the custom tags...
		metricTags := metric.Tags.ExpandTokens(false, env)

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

		metricDefs[i] = hmetrics.MetricDefinition{
			Tenant: endpoint.Tenant,
			Type:   metricType,
			ID:     os.Expand(mcm.Config.Collector.Metric_ID_Prefix, mappingFuncWithEnv) + os.Expand(metric.ID, mappingFunc),
			Tags:   map[string]string(allMetricTags),
		}
	}

	log.Tracef("Metric definitions being declared for endpoint: %v", endpoint.String())

	mcm.metricDefsChan <- metricDefs
}
