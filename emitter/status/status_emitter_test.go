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

package status

import (
	"fmt"
	"strings"
	"testing"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
	"github.com/hawkular/hawkular-openshift-agent/collector/tracker"
)

func TestJustAMessage(t *testing.T) {
	pod1_e1 := collector.CollectorID{PodID: "pod1", EndpointID: "pod1-e1"}
	mt := tracker.NewMetricsTracker(5)

	InitStatusReport("foo", "1", "aaabbb", 3)
	StatusReport.SetMetricsTracker(&mt)

	mt.AddMetricsFromCollector(pod1_e1, nil) // don't add any metrics
	StatusReport.SetEndpointMessage(pod1_e1, "SOMETHING")
	if _, ok := StatusReport.GetEndpointMessage(pod1_e1); !ok {
		t.Fatalf("pod1-e1 message should have been added")
	}

	StatusReport.Marshal() // Marshal performs the cleanup
	if _, ok := StatusReport.GetEndpointMessage(pod1_e1); !ok {
		t.Fatalf("pod1-e1 should still be there")
	}
}

func TestCleanupWhenRemovingEndpoints(t *testing.T) {
	pod1_e1 := collector.CollectorID{PodID: "pod1", EndpointID: "pod1-e1"}
	pod1_e2 := collector.CollectorID{PodID: "pod1", EndpointID: "pod1-e2"}
	pod2_e1 := collector.CollectorID{PodID: "pod2", EndpointID: "pod2-e1"}
	pod2_e2 := collector.CollectorID{PodID: "pod2", EndpointID: "pod2-e2"}
	X_X := collector.CollectorID{PodID: "X", EndpointID: "X"}

	mt := tracker.NewMetricsTracker(5)
	mt.AddMetricsFromCollector(pod1_e1, []hmetrics.MetricHeader{aMetric("a")})
	mt.AddMetricsFromCollector(pod1_e2, []hmetrics.MetricHeader{aMetric("b")})
	mt.AddMetricsFromCollector(pod2_e1, []hmetrics.MetricHeader{aMetric("c")})
	mt.AddMetricsFromCollector(pod2_e2, []hmetrics.MetricHeader{aMetric("d")})

	InitStatusReport("foo", "1", "aaabbb", 3)
	StatusReport.SetMetricsTracker(&mt)
	StatusReport.SetEndpointMessage(pod1_e1, "OK")
	StatusReport.SetEndpointMessage(pod1_e2, "OK")
	StatusReport.SetEndpointMessage(pod2_e1, "OK")
	StatusReport.SetEndpointMessage(pod2_e2, "OK")
	StatusReport.SetEndpointMessage(X_X, "OK")

	StatusReport.Marshal() // Marshal performs the cleanup
	if _, ok := StatusReport.GetEndpointMessage(pod1_e1); !ok {
		t.Fatalf("pod1-e1 should still be there")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod1_e2); !ok {
		t.Fatalf("pod1-e2 should still be there")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod2_e1); !ok {
		t.Fatalf("pod2-e1 should still be there")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod2_e2); !ok {
		t.Fatalf("pod2-e2 should still be there")
	}
	if _, ok := StatusReport.GetEndpointMessage(X_X); ok {
		t.Fatalf("unused should have been deleted from the cleanup")
	}

	mt.PurgeMetricsForCollectorEndpoint(collector.CollectorID{PodID: "pod1", EndpointID: "pod1-e1"})
	StatusReport.Marshal()
	if _, ok := StatusReport.GetEndpointMessage(pod1_e1); ok {
		t.Fatalf("pod1-e1 should have been deleted from the cleanup")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod1_e2); !ok {
		t.Fatalf("pod1-e2 should still be there")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod2_e1); !ok {
		t.Fatalf("pod2-e1 should still be there")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod2_e2); !ok {
		t.Fatalf("pod2-e2 should still be there")
	}

	mt.PurgeMetricsForCollectorEndpoint(collector.CollectorID{PodID: "pod1", EndpointID: "pod1-e2"})
	StatusReport.Marshal()
	if _, ok := StatusReport.GetEndpointMessage(pod1_e2); ok {
		t.Fatalf("pod1-e2 should have been deleted from the cleanup")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod2_e1); !ok {
		t.Fatalf("pod2-e1 should still be there")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod2_e2); !ok {
		t.Fatalf("pod2-e2 should still be there")
	}

	mt.PurgeMetricsForCollectorEndpoint(collector.CollectorID{PodID: "pod2", EndpointID: "pod2-e1"})
	StatusReport.Marshal()
	if _, ok := StatusReport.GetEndpointMessage(pod2_e1); ok {
		t.Fatalf("pod2-e1 should have been deleted from the cleanup")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod2_e2); !ok {
		t.Fatalf("pod2-e2 should still be there")
	}

	mt.PurgeMetricsForCollectorEndpoint(collector.CollectorID{PodID: "pod2", EndpointID: "pod2-e2"})
	StatusReport.Marshal()
	if _, ok := StatusReport.GetEndpointMessage(pod2_e2); ok {
		t.Fatalf("pod2-e2 should have been deleted from the cleanup")
	}

	if len(StatusReport.Endpoints) != 0 || len(mt.GetAllPods()) != 0 {
		t.Fatalf("All endpoints should have been deleted from the cleanup")
	}
}

func TestCleanupWhenThereAreNoMetrics(t *testing.T) {
	// test cleaning up when there are 0 metrics associated with pod/endpoint
	pod1_e1 := collector.CollectorID{PodID: "pod1", EndpointID: "pod1-e1"}
	pod2_e1 := collector.CollectorID{PodID: "pod2", EndpointID: "pod2-e1"}

	mt := tracker.NewMetricsTracker(5)

	mt.AddMetricsFromCollector(pod1_e1, nil)
	mt.AddMetricsFromCollector(pod2_e1, []hmetrics.MetricHeader{})

	InitStatusReport("foo", "1", "aaabbb", 3)
	StatusReport.SetMetricsTracker(&mt)
	StatusReport.SetEndpointMessage(pod1_e1, "OK")
	StatusReport.SetEndpointMessage(pod2_e1, "OK")

	StatusReport.Marshal() // Marshal performs the cleanup
	if _, ok := StatusReport.GetEndpointMessage(pod1_e1); !ok {
		t.Fatalf("pod1-e1 should still be there")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod2_e1); !ok {
		t.Fatalf("pod2-e1 should still be there")
	}

	mt.PurgeMetricsForCollectorEndpoint(pod1_e1)
	StatusReport.Marshal()
	if _, ok := StatusReport.GetEndpointMessage(pod1_e1); ok {
		t.Fatalf("pod1-e1 should have been deleted from the cleanup")
	}
	if _, ok := StatusReport.GetEndpointMessage(pod2_e1); !ok {
		t.Fatalf("pod2-e1 should still be there")
	}

	mt.PurgeMetricsForCollectorEndpoint(pod2_e1)
	StatusReport.Marshal()
	if _, ok := StatusReport.GetEndpointMessage(pod2_e1); ok {
		t.Fatalf("pod2-e1 should have been deleted from the cleanup")
	}

	if len(StatusReport.Endpoints) != 0 || len(mt.GetAllPods()) != 0 {
		t.Fatalf("All endpoints should have been deleted from the cleanup")
	}
}

func TestStatusReportEndpoints(t *testing.T) {
	pod1_e1 := collector.CollectorID{PodID: "pod1", EndpointID: "pod1-e1"}
	pod2_e1 := collector.CollectorID{PodID: "pod2", EndpointID: "pod2-e1"}

	InitStatusReport("foo", "1", "aaabbb", 3)
	if len(StatusReport.Endpoints) != 0 {
		t.Fatalf("endpoints did not initialize correctly")
	}

	if _, ok := StatusReport.GetEndpointMessage(pod1_e1); ok {
		t.Fatalf("should not have existed")
	}

	StatusReport.SetEndpointMessage(pod1_e1, "msg1")
	if e, ok := StatusReport.GetEndpointMessage(pod1_e1); e != "msg1" || !ok {
		t.Fatalf("failed to set endpoint")
	}

	StatusReport.SetEndpointMessage(pod2_e1, "msg2")
	if e, ok := StatusReport.GetEndpointMessage(pod2_e1); e != "msg2" || !ok {
		t.Fatalf("failed to set endpoint")
	}

	if len(StatusReport.Endpoints) != 2 {
		t.Fatalf("endpoints length not correct")
	}

	// delete them one by one until empty
	StatusReport.SetEndpointMessage(pod1_e1, "")
	if len(StatusReport.Endpoints) != 1 {
		t.Fatalf("endpoints length not correct")
	}

	StatusReport.SetEndpointMessage(pod2_e1, "")
	if len(StatusReport.Endpoints) != 0 {
		t.Fatalf("endpoints length not correct")
	}

	StatusReport.SetEndpointMessage(pod1_e1, "msg1")
	StatusReport.SetEndpointMessage(pod2_e1, "msg2")
	StatusReport.DeleteAllEndpointMessages()
	if len(StatusReport.Endpoints) != 0 {
		t.Fatalf("delete-all failed")
	}

}

func TestStatusReportRollingLog(t *testing.T) {
	InitStatusReport("foo", "1", "aaabbb", 3)

	if len(StatusReport.Log) != 3 {
		t.Fatalf("log did not initialize correctly")
	}

	StatusReport.AddLogMessage("one")
	if StatusReport.Log[0] != "" ||
		StatusReport.Log[1] != "" ||
		!strings.HasSuffix(StatusReport.Log[2], "one") {
		t.Fatalf("rolling log is bad: [%v]", StatusReport.Log)
	}

	StatusReport.AddLogMessage("two")
	if StatusReport.Log[0] != "" ||
		!strings.HasSuffix(StatusReport.Log[1], "one") ||
		!strings.HasSuffix(StatusReport.Log[2], "two") {
		t.Fatalf("rolling log is bad: [%v]", StatusReport.Log)
	}

	StatusReport.AddLogMessage("three")
	if !strings.HasSuffix(StatusReport.Log[0], "one") ||
		!strings.HasSuffix(StatusReport.Log[1], "two") ||
		!strings.HasSuffix(StatusReport.Log[2], "three") {
		t.Fatalf("rolling log is bad: [%v]", StatusReport.Log)
	}

	StatusReport.AddLogMessage("four")
	if !strings.HasSuffix(StatusReport.Log[0], "two") ||
		!strings.HasSuffix(StatusReport.Log[1], "three") ||
		!strings.HasSuffix(StatusReport.Log[2], "four") {
		t.Fatalf("rolling log is bad: [%v]", StatusReport.Log)
	}
}

func TestStatusReportConcurrency(t *testing.T) {
	gofuncs := 200
	gofuncChan := make(chan bool)
	for i := 0; i < gofuncs; i++ {
		go func(x int) {
			endpt := fmt.Sprintf("e%v", x)
			StatusReport.SetEndpointMessage(collector.CollectorID{PodID: endpt, EndpointID: endpt}, endpt)
			// These cause panics due to concurrent writes.
			// This illustrates why we don't use this and we use the Set methods instead
			//StatusReport.Endpoints[pod] = pod
			gofuncChan <- true
		}(i)
	}

	done := 0
	for _ = range gofuncChan {
		done++
		if done >= gofuncs {
			fmt.Printf("All [%v] go funcs are done\n", done)
			break
		}
	}

	if len(StatusReport.Endpoints) != gofuncs {
		t.Fatalf("Not all endpoints were added: %v", StatusReport.Endpoints)
	}
}

func aMetric(id string) hmetrics.MetricHeader {
	return hmetrics.MetricHeader{
		ID:     id,
		Tenant: "dummy-tenant",
		Type:   hmetrics.Gauge,
	}
}
