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
)

func TestCleanup(t *testing.T) {
	InitStatusReport("foo", "1", "aaabbb", 3)
	StatusReport.SetPod("pod1", []string{"pod1-e1", "pod1-e2"})
	StatusReport.SetPod("pod2", []string{"pod2-e1", "pod2-e2"})
	StatusReport.SetEndpoint("pod1-e1", "OK")
	StatusReport.SetEndpoint("pod1-e2", "OK")
	StatusReport.SetEndpoint("pod2-e1", "OK")
	StatusReport.SetEndpoint("pod2-e2", "OK")
	StatusReport.SetEndpoint("unused", "OK")

	StatusReport.Marshal() // shows that Marshal performs the cleanup
	if _, ok := StatusReport.GetEndpoint("pod1-e1"); !ok {
		t.Error("pod1-e1 should still be there")
	}
	if _, ok := StatusReport.GetEndpoint("pod1-e2"); !ok {
		t.Error("pod1-e2 should still be there")
	}
	if _, ok := StatusReport.GetEndpoint("pod2-e1"); !ok {
		t.Error("pod2-e1 should still be there")
	}
	if _, ok := StatusReport.GetEndpoint("pod2-e2"); !ok {
		t.Error("pod2-e2 should still be there")
	}
	if _, ok := StatusReport.GetEndpoint("unused"); ok {
		t.Error("unused should have been deleted from the cleanup")
	}

	StatusReport.SetPod("pod1", nil) // removing pod will do the cleanup of its endpoints
	if _, ok := StatusReport.GetEndpoint("pod1-e1"); ok {
		t.Error("pod1-e1 should have been deleted from the cleanup")
	}
	if _, ok := StatusReport.GetEndpoint("pod1-e2"); ok {
		t.Error("pod1-e2 should have been deleted from the cleanup")
	}
	if _, ok := StatusReport.GetEndpoint("pod2-e1"); !ok {
		t.Error("pod2-e1 should still be there")
	}
	if _, ok := StatusReport.GetEndpoint("pod2-e2"); !ok {
		t.Error("pod2-e2 should still be there")
	}

	StatusReport.SetPod("pod2", nil) // removing pod will do the cleanup of its endpoints
	if _, ok := StatusReport.GetEndpoint("pod2-e1"); ok {
		t.Error("pod2-e1 should have been deleted from the cleanup")
	}
	if _, ok := StatusReport.GetEndpoint("pod2-e2"); ok {
		t.Error("pod2-e2 should have been deleted from the cleanup")
	}

	if len(StatusReport.Endpoints) != 0 || len(StatusReport.Pods) != 0 {
		t.Error("All endpoints should have been deleted from the cleanup")
	}

	// When agent is collecting from external (non-OpenShift) endpoints, the IDs are prefixed with "X|X|".
	// In this case, the endpoints should never be cleaned up.
	StatusReport.SetEndpoint("X|X|e1", "OK")
	StatusReport.SetEndpoint("X|X|e2", "OK")
	StatusReport.cleanup(true)
	if _, ok := StatusReport.GetEndpoint("X|X|e1"); !ok {
		t.Error("e1 should still be there")
	}
	if _, ok := StatusReport.GetEndpoint("X|X|e2"); !ok {
		t.Error("e2 should still be there")
	}
}

func TestStatusReportPods(t *testing.T) {
	endpts1 := []string{"e1-a", "e1-b"}
	endpts2 := []string{"e2-a", "e2-b"}

	InitStatusReport("foo", "1", "aaabbb", 3)
	if len(StatusReport.Pods) != 0 {
		t.Error("pods did not initialize correctly")
	}

	if _, ok := StatusReport.GetPod("does-not-exist"); ok {
		t.Error("should not have existed")
	}

	StatusReport.SetPod("pod1", endpts1)
	if e, ok := StatusReport.GetPod("pod1"); e[0] != endpts1[0] || e[1] != endpts1[1] || !ok {
		t.Error("failed to set pod")
	}

	StatusReport.SetPod("pod2", endpts2)
	if e, ok := StatusReport.GetPod("pod2"); e[0] != endpts2[0] || e[1] != endpts2[1] || !ok {
		t.Error("failed to set pod")
	}

	if len(StatusReport.Pods) != 2 {
		t.Error("pods length not correct")
	}

	// delete them one by one until empty
	StatusReport.SetPod("pod1", nil)
	if len(StatusReport.Pods) != 1 {
		t.Error("pods length not correct")
	}

	StatusReport.SetPod("pod2", nil)
	if len(StatusReport.Pods) != 0 {
		t.Error("pods length not correct")
	}
}

func TestStatusReportEndpoints(t *testing.T) {
	InitStatusReport("foo", "1", "aaabbb", 3)
	if len(StatusReport.Endpoints) != 0 {
		t.Error("endpoints did not initialize correctly")
	}

	if _, ok := StatusReport.GetEndpoint("does-not-exist"); ok {
		t.Error("should not have existed")
	}

	StatusReport.SetEndpoint("e1", "msg1")
	if e, ok := StatusReport.GetEndpoint("e1"); e != "msg1" || !ok {
		t.Error("failed to set endpoint")
	}

	StatusReport.SetEndpoint("e2", "msg2")
	if e, ok := StatusReport.GetEndpoint("e2"); e != "msg2" || !ok {
		t.Error("failed to set endpoint")
	}

	if len(StatusReport.Endpoints) != 2 {
		t.Error("endpoints length not correct")
	}

	// delete them one by one until empty
	StatusReport.SetEndpoint("e1", "")
	if len(StatusReport.Endpoints) != 1 {
		t.Error("endpoints length not correct")
	}

	StatusReport.SetEndpoint("e2", "")
	if len(StatusReport.Endpoints) != 0 {
		t.Error("endpoints length not correct")
	}

	StatusReport.SetEndpoint("e1", "msg1")
	StatusReport.SetEndpoint("e2", "msg2")
	StatusReport.SetEndpoint("e3", "msg3")
	StatusReport.DeleteAllEndpoints()
	if len(StatusReport.Endpoints) != 0 {
		t.Error("delete-all failed")
	}

}

func TestStatusReportRollingLog(t *testing.T) {
	InitStatusReport("foo", "1", "aaabbb", 3)

	if len(StatusReport.Log) != 3 {
		t.Error("log did not initialize correctly")
	}

	StatusReport.AddLogMessage("one")
	if StatusReport.Log[0] != "" ||
		StatusReport.Log[1] != "" ||
		!strings.HasSuffix(StatusReport.Log[2], "one") {
		t.Error("rolling log is bad: [%v]", StatusReport.Log)
	}

	StatusReport.AddLogMessage("two")
	if StatusReport.Log[0] != "" ||
		!strings.HasSuffix(StatusReport.Log[1], "one") ||
		!strings.HasSuffix(StatusReport.Log[2], "two") {
		t.Error("rolling log is bad: [%v]", StatusReport.Log)
	}

	StatusReport.AddLogMessage("three")
	if !strings.HasSuffix(StatusReport.Log[0], "one") ||
		!strings.HasSuffix(StatusReport.Log[1], "two") ||
		!strings.HasSuffix(StatusReport.Log[2], "three") {
		t.Error("rolling log is bad: [%v]", StatusReport.Log)
	}

	StatusReport.AddLogMessage("four")
	if !strings.HasSuffix(StatusReport.Log[0], "two") ||
		!strings.HasSuffix(StatusReport.Log[1], "three") ||
		!strings.HasSuffix(StatusReport.Log[2], "four") {
		t.Error("rolling log is bad: [%v]", StatusReport.Log)
	}
}

func TestStatusReportConcurrency(t *testing.T) {
	gofuncs := 200
	gofuncChan := make(chan bool)
	for i := 0; i < gofuncs; i++ {
		go func(x int) {
			endpt := fmt.Sprintf("e%v", x)
			pod := fmt.Sprintf("p%v", x)
			StatusReport.SetEndpoint(endpt, endpt)
			StatusReport.SetPod(pod, []string{pod})
			// These cause panics due to concurrent writes.
			// This illustrates why we don't use this and we use the Set methods instead
			//StatusReport.Endpoints[pod] = pod
			//StatusReport.Pods[endpt] = []string{endpt}
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

	if len(StatusReport.Pods) != gofuncs {
		t.Errorf("Not all pods were added: %v", StatusReport.Pods)
	}

	if len(StatusReport.Endpoints) != gofuncs {
		t.Errorf("Not all endpoints were added: %v", StatusReport.Endpoints)
	}
}
