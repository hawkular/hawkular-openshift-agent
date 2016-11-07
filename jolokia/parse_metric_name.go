/*
   Copyright 2016 Red Hat, Inc. and/or its affiliates
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

package jolokia

import (
	"fmt"
	"strings"
)

// ParseMetricName takes a string that describes a particular metric that is to be collected
// and splits it into the parts necessary for requesting the metric data from Jolokia.
//
// The parts are stored in the given request.
//
// The string must comprise of an MBean name, an attribute on that MBean, and an optional path
// that drills down further into a composite attribute to refer to the actual metric datapoint.
//
// Examples:
//   java.lang:type=Threading#ThreadCount
//   java.lang:type=Memory#HeapMemoryUsage#init
func ParseMetricName(metricName string, req *JolokiaRequest) (err error) {
	arr := strings.SplitN(metricName, "#", 3)
	switch len(arr) {
	case 2:
		req.MBean = arr[0]
		req.Attribute = arr[1]
	case 3:
		req.MBean = arr[0]
		req.Attribute = arr[1]
		req.Path = arr[2]
	default:
		err = fmt.Errorf("Bad metric name string [%v]", metricName)
	}
	return
}
