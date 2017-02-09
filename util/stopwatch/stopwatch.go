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

package stopwatch

import (
	"time"
)

// Stopwatch is a very simple utility that captures elapsed wall clock time.
type Stopwatch struct {
	start            time.Time
	lastMarkDuration time.Duration
}

// Creates a new Stopwatch and resets it to the current time. See Reset().
func NewStopwatch() Stopwatch {
	s := Stopwatch{}
	s.Reset()
	return s
}

// Reset sets the stopwatch to the current time and clears the last marked time.
func (s *Stopwatch) Reset() {
	s.start = time.Now()
	s.lastMarkDuration = 0
}

// MarkTime will capture the duration of time that has
// elapsed since the stopwatch was last reset and returns that duration.
// This value is also returned via LastMarkTime().
func (s *Stopwatch) MarkTime() time.Duration {
	s.lastMarkDuration = time.Since(s.start)
	return s.lastMarkDuration
}

// LastMartTime returns the duration last returned by MarkTime().
func (s *Stopwatch) LastMarkTime() time.Duration {
	return s.lastMarkDuration
}

// StartTime returns when the stopwatch started or was last reset.
func (s *Stopwatch) StartTime() time.Time {
	return s.start
}

// String returns the last marked duration as a string.
func (s Stopwatch) String() string {
	return s.lastMarkDuration.String()
}
