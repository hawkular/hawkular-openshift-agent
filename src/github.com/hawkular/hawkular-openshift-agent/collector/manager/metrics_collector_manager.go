package manager

import (
	"sync"
	"time"

	"github.com/golang/glog"
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/collector/impl"
	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/config/tags"
	"github.com/hawkular/hawkular-openshift-agent/log"
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
			var theCollector collector.MetricsCollector
			id := e.Url
			switch e.Type {
			case collector.ENDPOINT_TYPE_PROMETHEUS:
				{
					theCollector = impl.NewPrometheusMetricsCollector(id, mcm.Config.Identity, e, nil)
				}
			case collector.ENDPOINT_TYPE_JOLOKIA:
				{
					theCollector = impl.NewJolokiaMetricsCollector(id, mcm.Config.Identity, e, nil)
				}
			default:
				{
					glog.Warningf("Will not start collecting for endpoint [%v] - unknown endpoint type [%v]", e.Url, e.Type)
					return
				}
			}

			mcm.StartCollecting(theCollector)
		}
	}
	return

}

// StartCollecting will collect metrics every "collection interval" seconds in a go routine.
// If a metrics collector with the same ID is already collecting metrics, it will be stopped
// and the given new collector will take its place.
func (mcm *MetricsCollectorManager) StartCollecting(collector collector.MetricsCollector) {
	id := collector.GetId()

	// if there was an old ticker still running for this collector, stop it
	mcm.StopCollecting(id)

	// lock access to the Tickers array
	mcm.TickersLock.Lock()
	defer mcm.TickersLock.Unlock()

	interval := collector.GetEndpoint().Collection_Interval_Secs
	if interval < mcm.Config.Collector.Minimum_Collection_Interval_Secs {
		glog.Warningf("Collection interval for [%v] is [%v] which is lower than the minimum allowed [%v]. Setting it to the minimum allowed.",
			id, interval, mcm.Config.Collector.Minimum_Collection_Interval_Secs)
		interval = mcm.Config.Collector.Minimum_Collection_Interval_Secs
	}

	// before we start collecting metrics, we need to declare the metric definitions
	mcm.declareMetricDefinitions(collector.GetEndpoint(), collector.GetAdditionalEnvironment())

	glog.Infof("START collecting metrics from [%v] every [%v]s", id, interval)
	ticker := time.NewTicker(time.Second * time.Duration(interval))
	mcm.Tickers[id] = ticker
	go func() {
		for _ = range ticker.C {
			metrics, err := collector.CollectMetrics()
			if err != nil {
				glog.Warningf("Failed to collect metrics from [%v]. err=%v", id, err)
			} else {
				mcm.metricsChan <- metrics
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
		glog.Infof("STOP collecting metrics from [%v]", collectorId)
		ticker.Stop()
		delete(mcm.Tickers, collectorId)
	}
}

// StopCollectingAll halts all metric collections.
func (mcm *MetricsCollectorManager) StopCollectingAll() {
	// lock access to the Tickers array
	mcm.TickersLock.Lock()
	defer mcm.TickersLock.Unlock()

	glog.Infof("STOP collecting all metrics from all endpoints")
	for id, ticker := range mcm.Tickers {
		ticker.Stop()
		delete(mcm.Tickers, id)
	}
}

func (mcm *MetricsCollectorManager) declareMetricDefinitions(endpoint *collector.Endpoint, additionalEnv map[string]string) {

	metricDefs := make([]hmetrics.MetricDefinition, len(endpoint.Metrics))

	// For each metric in the endpoint, create a metric def for it;
	// notice metric tags override endpoint tags which override global tags
	globalTags := mcm.Config.Tags.ExpandTokens(true, additionalEnv)
	endpointTags := endpoint.Tags.ExpandTokens(true, additionalEnv)

	for i, metric := range endpoint.Metrics {
		metricTags := metric.Tags.ExpandTokens(true, additionalEnv)

		allMetricTags := tags.Tags{}
		allMetricTags.AppendTags(globalTags)   // global tags are overridden by...
		allMetricTags.AppendTags(endpointTags) // endpoint tags which are overridden by...
		allMetricTags.AppendTags(metricTags)   // metric tags.

		metricDefs[i] = hmetrics.MetricDefinition{
			Tenant: endpoint.Tenant,
			Type:   metric.Type,
			ID:     metric.Id,
			Tags:   map[string]string(allMetricTags),
		}
	}

	log.Tracef("Metric definitions being declared for endpoint: %v", endpoint.String())

	mcm.metricDefsChan <- metricDefs
}
