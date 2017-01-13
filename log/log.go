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

package log

import (
	"fmt"

	"github.com/golang/glog"

	"github.com/hawkular/hawkular-openshift-agent/emitter/status"
)

const (
	debug glog.Level = glog.Level(4)
	trace glog.Level = glog.Level(5)
)

func Info(args ...interface{}) {
	glog.InfoDepth(1, args...)
}

func Infof(format string, args ...interface{}) {
	glog.InfoDepth(1, fmt.Sprintf(format, args...))
}

func Warning(args ...interface{}) {
	glog.WarningDepth(1, args...)
	status.StatusReport.AddLogMessage("WARNING: " + fmt.Sprint(args...))
}

func Warningf(format string, args ...interface{}) {
	glog.WarningDepth(1, fmt.Sprintf(format, args...))
	status.StatusReport.AddLogMessage(fmt.Sprintf("WARNING: "+format, args...))
}

func Error(args ...interface{}) {
	glog.ErrorDepth(1, args...)
	status.StatusReport.AddLogMessage("ERROR: " + fmt.Sprint(args...))
}

func Errorf(format string, args ...interface{}) {
	glog.ErrorDepth(1, fmt.Sprintf(format, args...))
	status.StatusReport.AddLogMessage(fmt.Sprintf("ERROR: "+format, args...))
}

// Debug will log a message at verbose level 4 and will ensure the caller's stack frame is used
func Debug(args ...interface{}) {
	if glog.V(debug) {
		glog.InfoDepth(1, "DEBUG: "+fmt.Sprint(args...)) // 1 == depth in the stack of the caller
	}
}

// Debugf will log a message at verbose level 4 and will ensure the caller's stack frame is used
func Debugf(format string, args ...interface{}) {
	if glog.V(debug) {
		glog.InfoDepth(1, fmt.Sprintf("DEBUG: "+format, args...)) // 1 == depth in the stack of the caller
	}
}

func IsDebug() bool {
	return bool(glog.V(debug))
}

// Trace will log a message at verbose level 5 and will ensure the caller's stack frame is used
func Trace(args ...interface{}) {
	if glog.V(trace) {
		glog.InfoDepth(1, "TRACE: "+fmt.Sprint(args...)) // 1 == depth in the stack of the caller
	}
}

// Tracef will log a message at verbose level 5 and will ensure the caller's stack frame is used
func Tracef(format string, args ...interface{}) {
	if glog.V(trace) {
		glog.InfoDepth(1, fmt.Sprintf("TRACE: "+format, args...)) // 1 == depth in the stack of the caller
	}
}

func IsTrace() bool {
	return bool(glog.V(trace))
}
