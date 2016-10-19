package storage

import (
	"github.com/golang/glog"
	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/config"
	"github.com/hawkular/hawkular-openshift-agent/log"
)

type MetricsStorageManager struct {
	MetricsChannel chan []hmetrics.MetricHeader
	hawkularClient *hmetrics.Client
	globalConfig   *config.Config
}

func NewMetricsStorageManager(conf *config.Config) (ms *MetricsStorageManager, err error) {
	client, err := getHawkularMetricsClient(conf)
	if err != nil {
		return nil, err
	}

	ms = &MetricsStorageManager{
		MetricsChannel: make(chan []hmetrics.MetricHeader, 100),
		hawkularClient: client,
		globalConfig:   conf,
	}
	return
}

func (ms *MetricsStorageManager) StartStoringMetrics() {
	glog.Info("START storing metrics")
	go ms.consumeMetrics()
}

func (ms *MetricsStorageManager) StopStoringMetrics() {
	glog.Info("STOP storing metrics")
	close(ms.MetricsChannel)
}

func (ms *MetricsStorageManager) consumeMetrics() {
	for metrics := range ms.MetricsChannel {
		if len(metrics) == 0 {
			continue
		}

		// If a tenant is provided, use it. Otherwise, use the global tenant.
		// This assumes all metrics in the given array are associated with the same tenant.
		var tenant string
		if metrics[0].Tenant != "" {
			tenant = metrics[0].Tenant
		} else {
			tenant = ms.globalConfig.Hawkular_Server.Tenant
		}

		// Store the metrics to H-Metrics.
		err := ms.hawkularClient.Write(metrics, hmetrics.Tenant(tenant))

		if err != nil {
			glog.Warningf("Failed to store metrics. err=%v", err)
		} else {
			log.Debugf("Stored datapoints for [%v] metrics", len(metrics))
			if log.IsTrace() {
				for _, m := range metrics {
					log.Tracef("Stored [%v] [%v] datapoints for metric named [%v]: %v", len(m.Data), m.Type, m.ID, m.Data)
				}
			}
		}
	}
}

func getHawkularMetricsClient(conf *config.Config) (*hmetrics.Client, error) {
	params := hmetrics.Parameters{
		Tenant:   conf.Hawkular_Server.Tenant,
		Url:      conf.Hawkular_Server.Url,
		Username: conf.Hawkular_Server.Username,
		Password: conf.Hawkular_Server.Password,
		Token:    conf.Hawkular_Server.Token,
	}

	return hmetrics.NewHawkularClient(params)
}
