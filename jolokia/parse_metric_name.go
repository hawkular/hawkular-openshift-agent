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

package jolokia

import (
	"fmt"
	"strings"
)

// ParseMetricNameForJolokiaRequest takes a string that describes a particular metric that is to be collected
// and splits it into the parts necessary for requesting the metric data from Jolokia. The metric name split
// into its parts is returned as is any error that occurred while parsing the metric name.
//
// The parts are stored in the given request.
//
// The string must comprise of an MBean name, an attribute on that MBean, and an optional path
// that drills down further into a composite attribute to refer to the actual metric datapoint.
//
// The metric name can include wildcard patterns - for more on that see NewMetricNameParts.
func ParseMetricNameForJolokiaRequest(metricName string, req *JolokiaRequest) (MetricNameParts, error) {
	parts, err := NewMetricNameParts(metricName)
	if err == nil {
		req.MBean = parts.MBean
		if parts.HasAttribute() && !parts.IsAllAttributes() {
			req.Attribute = parts.Attribute
		}
		if parts.HasPath() && !parts.IsAllPaths() {
			req.Path = parts.Path
		}
	}
	return parts, err
}

type MetricNameParts struct {
	MBean        string
	Attribute    string
	Path         string
	KeyValues    map[string]string // the object name key and their values (which may or may not be "*")
	KeyWildcards map[string]bool   // those object name keys whose values were "*"
}

// NewMetricNameParts takes a string that describes a particular metric that is to be collected
// and parses its different parts returning an object that describes those parts.
// The string must comprise of an MBean name, an optional attribute on that MBean, and an optional path
// that drills down further into a composite attribute to refer to the actual metric datapoint.
// MBean name keys can have wildfly values ("*").
// Attributes and the paths may also be wildcard values ("*").
// If Attribute is not specified, it is assumed "*".
//
// Examples:
//   java.lang:type=Threading#ThreadCount
//   java.lang:type=*#*
//   java.lang:type=Memory#HeapMemoryUsage#init
//   java.lang:type=Memory#*#*
//   java.lang:type=*#*#*
//
// Note that MBean name patterns must include a key - wildcards without a key is not supported.
// For example, this is unsupported: "java.lang:*"
// Wildcards for domains are also not supported (i.e. "*:type=Memory").
func NewMetricNameParts(metricName string) (parts MetricNameParts, err error) {
	parts = MetricNameParts{
		KeyValues:    make(map[string]string, 0),
		KeyWildcards: make(map[string]bool, 0),
	}

	arr := strings.SplitN(metricName, "#", 3)
	switch len(arr) {
	case 1:
		parts.MBean = arr[0]
		parts.Attribute = "*"
	case 2:
		parts.MBean = arr[0]
		parts.Attribute = arr[1]
	case 3:
		parts.MBean = arr[0]
		parts.Attribute = arr[1]
		parts.Path = arr[2]
	default:
		return parts, fmt.Errorf("Bad metric name string [%v]", metricName)
	}

	// For all the wildcard patterns in the mbean name, determine which keys they are for.
	// First, we don't care about the domain (the left of the ":") so ignore that part.
	// Second, key-value pairs are comma-separated.
	// Third, key-value pairs are in the form "key=value" where value may be the wildcard "*".
	// If "key=value" is a "*" (that is both key and value are undetermined) this is unsupported and an error.
	arr = strings.SplitN(parts.MBean, ":", 2)
	var keyValuePairs string
	if len(arr) == 1 {
		keyValuePairs = arr[0]
	} else {
		keyValuePairs = arr[1]
	}

	for _, keyValuePair := range strings.Split(keyValuePairs, ",") {
		keyValuePair = strings.TrimSpace(keyValuePair)
		if keyValuePair == "*" {
			return parts, fmt.Errorf("Must supply a key name for wildcard expression in metric name [%v]", metricName)
		}
		keyValueArr := strings.SplitN(keyValuePair, "=", 2)
		if len(keyValueArr) != 2 {
			return parts, fmt.Errorf("[%v] in metric name [%v] is invalid - must be 'key=value' format.", keyValuePair, metricName)
		} else {
			parts.KeyValues[keyValueArr[0]] = keyValueArr[1]
			if keyValueArr[1] == "*" {
				parts.KeyWildcards[keyValueArr[0]] = true
			}
		}
	}

	return
}

func (parts *MetricNameParts) HasAttribute() bool {
	return parts.Attribute != ""
}

func (parts *MetricNameParts) HasPath() bool {
	return parts.Path != ""
}

func (parts *MetricNameParts) IsAllAttributes() bool {
	return parts.Attribute == "*"
}

func (parts *MetricNameParts) IsAllPaths() bool {
	return parts.Path == "*" || parts.IsAllAttributes()
}

func (parts *MetricNameParts) IsMBeanQuery() bool {
	return len(parts.KeyWildcards) > 0
}
