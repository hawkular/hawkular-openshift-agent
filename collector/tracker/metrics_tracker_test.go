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
package tracker

import (
	"testing"

	hmetrics "github.com/hawkular/hawkular-client-go/metrics"

	"github.com/hawkular/hawkular-openshift-agent/collector"
)

func TestAddZeroMetrics(t *testing.T) {
	mtO := NewMetricsTracker(1) // MAX ALLOWED IS 1 PER POD
	mt := &mtO

	if len(mt.GetAllPods()) != 0 {
		t.Fatalf("Should have started empty")
	}

	cid := collector.CollectorID{PodID: "pid1", EndpointID: "eid1"}

	// It is possible to add 0 metrics but still track a pod
	o := mt.AddMetricsFromCollector(cid, nil)
	if GetMetricCountForCollectorEndpoint(mt, cid) != 0 {
		t.Fatalf("Bad metric count - should have zero metrics")
	}
	if o != nil {
		t.Fatalf("Should not have overflowed since we didn't add anything: %v", o)
	}
	if len(mt.GetAllPods()) != 1 {
		t.Fatalf("Should have a pod: %v", mt.GetAllPods())
	}
}

func TestAddDuplicate(t *testing.T) {
	mtO := NewMetricsTracker(1) // MAX ALLOWED IS 1 PER POD
	mt := &mtO

	cid := collector.CollectorID{PodID: "pid1", EndpointID: "eid1"}

	// It is possible to add multiple metrics that have the same name
	o := mt.AddMetricsFromCollector(cid, []hmetrics.MetricHeader{aMetric("a"), aMetric("a")})
	if GetMetricCountForCollectorEndpoint(mt, cid) != 1 {
		t.Fatalf("Bad metric count - should only have one metric")
	}
	if o != nil {
		t.Fatalf("Should not have overflowed since all metrics had the same ID: %v", o)
	}
}

func TestAdd(t *testing.T) {
	mtO := NewMetricsTracker(5) // MAX ALLOWED IS 5 PER POD
	mt := &mtO

	cid := collector.CollectorID{PodID: "pid1", EndpointID: "eid1"}

	if GetMetricCountForCollectorPod(mt, cid) != 0 {
		t.Fatalf("Bad metric count - should have started empty")
	}

	mt.AddMetricsFromCollector(cid, nil)
	if GetMetricCountForCollectorPod(mt, cid) != 0 {
		t.Fatalf("Bad metric count - didn't add anything")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid) != 0 {
		t.Fatalf("Bad metric count - didn't add anything")
	}

	mt.AddMetricsFromCollector(cid, []hmetrics.MetricHeader{})
	if GetMetricCountForCollectorPod(mt, cid) != 0 {
		t.Fatalf("Bad metric count - didn't add anything")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid) != 0 {
		t.Fatalf("Bad metric count - didn't add anything")
	}

	mt.AddMetricsFromCollector(cid, []hmetrics.MetricHeader{aMetric("a")})
	if GetMetricCountForCollectorPod(mt, cid) != 1 {
		t.Fatalf("Bad metric count")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid) != 1 {
		t.Fatalf("Bad metric count")
	}

	mt.AddMetricsFromCollector(cid, []hmetrics.MetricHeader{aMetric("a")})
	if GetMetricCountForCollectorPod(mt, cid) != 1 {
		t.Fatalf("Bad metric count - we added the same metric so it should not have changed count")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid) != 1 {
		t.Fatalf("Bad metric count - we added the same metric so it should not have changed count")
	}

	mt.AddMetricsFromCollector(cid, []hmetrics.MetricHeader{aMetric("a"), aMetric("b")})
	if GetMetricCountForCollectorPod(mt, cid) != 2 {
		t.Fatalf("Bad metric count - we added one existing one but we did add one new one")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid) != 2 {
		t.Fatalf("Bad metric count - we added one existing one but we did add one new one")
	}

	mt.AddMetricsFromCollector(cid, []hmetrics.MetricHeader{aMetric("c"), aMetric("d")})
	if GetMetricCountForCollectorPod(mt, cid) != 4 {
		t.Fatalf("Bad metric count - we added two new ones")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid) != 4 {
		t.Fatalf("Bad metric count - we added two new ones")
	}

	mt.PurgeMetricsForCollectorEndpoint(cid)
	if GetMetricCountForCollectorPod(mt, cid) != 0 {
		t.Fatalf("Bad metric count - we should have purged")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid) != 0 {
		t.Fatalf("Bad metric count - we should have purged")
	}
}

func TestGetAll(t *testing.T) {
	mtO := NewMetricsTracker(5) // MAX ALLOWED IS 5 PER POD
	mt := &mtO

	cid0 := collector.CollectorID{PodID: "pid1", EndpointID: "eid1"}
	mt.AddMetricsFromCollector(cid0, []hmetrics.MetricHeader{aMetric("a")})
	mt.AddMetricsFromCollector(cid0, []hmetrics.MetricHeader{aMetric("b")})
	if len(mt.GetAllPods()) != 1 {
		t.Fatalf("Should be one pod: %v", mt.GetAllPods())
	}
	if _, ok := mt.GetAllPods()["pid1"]; !ok {
		t.Fatalf("Bad pod id: %v", mt.GetAllPods())
	}
	if len(mt.GetAllEndpoints("pid1")) != 1 {
		t.Fatalf("Should be one endpoint in pod: %v", mt.GetAllEndpoints("pid1"))
	}
	if mt.GetAllEndpoints("pid1")["eid1"] != 2 {
		t.Fatalf("Should be one metric in endpoint in pod")
	}

	cid1 := collector.CollectorID{PodID: "pid2", EndpointID: "eid1"}
	cid2 := collector.CollectorID{PodID: "pid2", EndpointID: "eid2"}
	mt.AddMetricsFromCollector(cid1, []hmetrics.MetricHeader{aMetric("a")})
	mt.AddMetricsFromCollector(cid1, []hmetrics.MetricHeader{aMetric("b")})
	mt.AddMetricsFromCollector(cid2, []hmetrics.MetricHeader{aMetric("c")})
	if len(mt.GetAllPods()) != 2 {
		t.Fatalf("Should be two pods: %v", mt.GetAllPods())
	}
	if len(mt.GetAllEndpoints("pid2")) != 2 {
		t.Fatalf("Should be two endpoints in pod: %v", mt.GetAllEndpoints("pid2"))
	}
	if mt.GetAllEndpoints("pid2")["eid1"] != 2 {
		t.Fatalf("Should be two metrics in endpoint in pod")
	}
	if mt.GetAllEndpoints("pid2")["eid2"] != 1 {
		t.Fatalf("Should be one metric in endpoint in pod")
	}

	mt.PurgeMetricsForCollectorEndpoint(cid1)
	if len(mt.GetAllPods()) != 2 {
		t.Fatalf("Should still be two pods after endpoint purge: %v", mt.GetAllPods())
	}

	mt.PurgeMetricsForCollectorEndpoint(cid2)
	if len(mt.GetAllPods()) != 1 {
		t.Fatalf("Should be one more pod after purge: %v", mt.GetAllPods())
	}

	mt.PurgeMetricsForCollectorEndpoint(cid0)
	if len(mt.GetAllPods()) != 0 {
		t.Fatalf("Should be no more pods after purge: %v", mt.GetAllPods())
	}
}

func TestOverflow(t *testing.T) {
	mtO := NewMetricsTracker(5) // MAX ALLOWED IS 5 PER POD
	mt := &mtO

	cid1_1 := collector.CollectorID{PodID: "pid1", EndpointID: "eid1"}
	cid1_2 := collector.CollectorID{PodID: "pid1", EndpointID: "eid2"}
	cid1_3 := collector.CollectorID{PodID: "pid1", EndpointID: "eid3"}
	cid2_1 := collector.CollectorID{PodID: "pid2", EndpointID: "eid1"}
	cid2_2 := collector.CollectorID{PodID: "pid2", EndpointID: "eid2"}
	cid2_3 := collector.CollectorID{PodID: "pid2", EndpointID: "eid3"}

	o1 := mt.AddMetricsFromCollector(cid1_1, []hmetrics.MetricHeader{aMetric("a"), aMetric("b")})
	if o1 != nil {
		t.Fatalf("Should not have overflowed here")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid1_1) != 2 {
		t.Fatalf("Bad metric count for endpoint 1_1")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid1_2) != 0 {
		t.Fatalf("Bad metric count for endpoint 1_2")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid1_3) != 0 {
		t.Fatalf("Bad metric count for endpoint 1_3")
	}
	if pc, ec, mc := mt.GetCounts(); pc != 1 || ec != 1 || mc != 2 {
		t.Fatalf("Bad counts pc=%v, ec=%v, mc=%v", pc, ec, mc)
	}

	o2 := mt.AddMetricsFromCollector(cid1_2, []hmetrics.MetricHeader{aMetric("c"), aMetric("d")})
	if o2 != nil {
		t.Fatalf("Should not have overflowed here")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid1_1) != 2 {
		t.Fatalf("Bad metric count for endpoint 1_1")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid1_2) != 2 {
		t.Fatalf("Bad metric count for endpoint 1_2")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid1_3) != 0 {
		t.Fatalf("Bad metric count for endpoint 1_3")
	}
	if pc, ec, mc := mt.GetCounts(); pc != 1 || ec != 2 || mc != 4 {
		t.Fatalf("Bad counts pc=%v, ec=%v, mc=%v", pc, ec, mc)
	}

	o3 := mt.AddMetricsFromCollector(cid1_3, []hmetrics.MetricHeader{aMetric("e"), aMetric("f")})
	if len(o3) != 1 {
		t.Fatalf("Should have overflowed by 1 metric: %v\n%v", o3, mt.String())
	}
	if o3[0].ID != "f" {
		t.Fatalf("Should have overflowed the f metric: %v", o3[0])
	}
	if GetMetricCountForCollectorEndpoint(mt, cid1_1) != 2 {
		t.Fatalf("Bad metric count for endpoint 1_1")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid1_2) != 2 {
		t.Fatalf("Bad metric count for endpoint 1_2")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid1_3) != 1 {
		t.Fatalf("Bad metric count for endpoint 1_3")
	}
	if pc, ec, mc := mt.GetCounts(); pc != 1 || ec != 3 || mc != 5 {
		t.Fatalf("Bad counts pc=%v, ec=%v, mc=%v", pc, ec, mc)
	}

	if len(mt.GetAllPods()) != 1 {
		t.Fatalf("Wrong pods: %v", mt.GetAllPods())
	}

	// add for another pod
	if GetMetricCountForCollectorEndpoint(mt, cid2_1) != 0 {
		t.Fatalf("Bad metric count for endpoint 2_1")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_2) != 0 {
		t.Fatalf("Bad metric count for endpoint 2_2")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_3) != 0 {
		t.Fatalf("Bad metric count for endpoint 2_3")
	}
	o4 := mt.AddMetricsFromCollector(cid2_1, []hmetrics.MetricHeader{aMetric("a"), aMetric("b")})
	if o4 != nil {
		t.Fatalf("Should not have overflowed here")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_1) != 2 {
		t.Fatalf("Bad metric count for endpoint 2_1")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_2) != 0 {
		t.Fatalf("Bad metric count for endpoint 2_2")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_3) != 0 {
		t.Fatalf("Bad metric count for endpoint 2_3")
	}
	if pc, ec, mc := mt.GetCounts(); pc != 2 || ec != 4 || mc != 7 {
		t.Fatalf("Bad counts pc=%v, ec=%v, mc=%v", pc, ec, mc)
	}

	o5 := mt.AddMetricsFromCollector(cid2_2, []hmetrics.MetricHeader{aMetric("c"), aMetric("d")})
	if o5 != nil {
		t.Fatalf("Should not have overflowed here")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_1) != 2 {
		t.Fatalf("Bad metric count for endpoint 2_1")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_2) != 2 {
		t.Fatalf("Bad metric count for endpoint 2_2")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_3) != 0 {
		t.Fatalf("Bad metric count for endpoint 2_3")
	}
	if pc, ec, mc := mt.GetCounts(); pc != 2 || ec != 5 || mc != 9 {
		t.Fatalf("Bad counts pc=%v, ec=%v, mc=%v", pc, ec, mc)
	}

	count := GetMetricCountForCollectorPod(mt, cid2_2)

	// if these were different metrics it would overflow, but we are adding "c" and "d" again on same endpoint
	o6 := mt.AddMetricsFromCollector(cid2_2, []hmetrics.MetricHeader{aMetric("c"), aMetric("d")})
	if o6 != nil {
		t.Fatalf("Should not have overflowed here: %v\n%v", o6, mt.String())
	}
	if count != GetMetricCountForCollectorPod(mt, cid2_2) {
		t.Fatalf("Metric count should not have changed - we added the same metrics")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_1) != 2 {
		t.Fatalf("Bad metric count for endpoint 2_1")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_2) != 2 {
		t.Fatalf("Bad metric count for endpoint 2_2")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_3) != 0 {
		t.Fatalf("Bad metric count for endpoint 2_3")
	}
	if pc, ec, mc := mt.GetCounts(); pc != 2 || ec != 5 || mc != 9 {
		t.Fatalf("Bad counts pc=%v, ec=%v, mc=%v", pc, ec, mc)
	}

	// these are different metrics (different endpoints) so it should overflow
	o7 := mt.AddMetricsFromCollector(cid2_3, []hmetrics.MetricHeader{aMetric("c"), aMetric("d")})
	if len(o7) != 1 {
		t.Fatalf("Should have overflowed here: %v\n%v", o7, mt.String())
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_1) != 2 {
		t.Fatalf("Bad metric count for endpoint 2_1")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_2) != 2 {
		t.Fatalf("Bad metric count for endpoint 2_2")
	}
	if GetMetricCountForCollectorEndpoint(mt, cid2_3) != 1 {
		t.Fatalf("Bad metric count for endpoint 2_3")
	}
	if pc, ec, mc := mt.GetCounts(); pc != 2 || ec != 6 || mc != 10 {
		t.Fatalf("Bad counts pc=%v, ec=%v, mc=%v", pc, ec, mc)
	}

	if len(mt.GetAllPods()) != 2 {
		t.Fatalf("Wrong pods: %v", mt.GetAllPods())
	}

	mt.PurgeAllMetrics()
	if len(mt.GetAllPods()) != 0 {
		t.Fatalf("Purge all failed: %v", mt.GetAllPods())
	}
	if pc, ec, mc := mt.GetCounts(); pc != 0 || ec != 0 || mc != 0 {
		t.Fatalf("Bad counts pc=%v, ec=%v, mc=%v", pc, ec, mc)
	}
}

func TestUnknownCount(t *testing.T) {
	mtO := NewMetricsTracker(5)
	mt := &mtO

	if GetMetricCountForCollectorEndpoint(mt, collector.CollectorID{PodID: "unknown", EndpointID: "unknown"}) != 0 {
		t.Fatalf("Should not have any metrics for an unknown endpoint")
	}
	if GetMetricCountForCollectorPod(mt, collector.CollectorID{PodID: "unknown", EndpointID: "unknown"}) != 0 {
		t.Fatalf("Should not have any metrics for an unknown collector")
	}
	if GetMetricCountForCollectorPod(mt, collector.CollectorID{PodID: "", EndpointID: "unknown"}) != 0 {
		t.Fatalf("Should not have any metrics for an unknown collector")
	}

	if len(mt.GetAllPods()) != 0 {
		t.Fatalf("GetAllPods should return nothing")
	}
	if len(mt.GetAllEndpoints("unknown")) != 0 {
		t.Fatalf("GetAllEndpoints should return nothing")
	}

	p, e, m := mt.GetCounts()
	if p != 0 || e != 0 || m != 0 {
		t.Fatalf("GetCounts should return all zeroes")
	}

	// make sure purging an empty tracker doesn't panic
	mt.PurgeAllMetrics()
	mt.PurgeMetricsForCollectorEndpoint(collector.CollectorID{PodID: "unknown", EndpointID: "unknown"})
}

func GetMetricCountForCollectorEndpoint(mt *MetricsTracker, cid collector.CollectorID) int {
	return mt.GetAllEndpoints(cid.PodID)[cid.EndpointID]
}

func GetMetricCountForCollectorPod(mt *MetricsTracker, cid collector.CollectorID) int {
	count := 0
	for _, mcount := range mt.GetAllEndpoints(cid.PodID) {
		count += mcount
	}
	return count
}

func aMetric(id string) hmetrics.MetricHeader {
	return hmetrics.MetricHeader{
		ID:     id,
		Tenant: "dummy-tenant",
		Type:   hmetrics.Gauge,
	}
}
