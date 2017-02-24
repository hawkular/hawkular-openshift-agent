/*
   Copyright 2017 Red Hat, Inc. and/or its affiliates
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

package main

import (
	"encoding/json"
	"expvar" // this has the side effect of registering the /debug/vars HTTP endpoint.
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

// Type to be an example of metric data in maps-o-maps.
// Outer key is HTTP method, inner key is request path, value is response time
type SimulatedHttpRequests map[string]map[string]float64

func (s SimulatedHttpRequests) String() string {
	b, _ := json.Marshal(s)
	return string(b)
}

// WaveDataType to be an example of metric data in an array
type WaveDataType struct {
	data []float64
}

func (w WaveDataType) String() string {
	b, _ := json.Marshal(w.data)
	return string(b)
}

// Command line arguments
var (
	argPort = flag.String("port", "8181", "The port to emit the expvar JSON data.")
)

// Some internals
var (
	animalTypes = []string{"Deer", "Goose", "Squirrel", "Turkey"}
	waveData    = WaveDataType{
		data: make([]float64, 100),
	}
	simulatedHttp = SimulatedHttpRequests(make(map[string]map[string]float64, 0))
)

// Exposed Data
var (
	animals    = expvar.NewMap("expvar.example.animals-map")
	loopCount  = expvar.NewFloat("expvar.example.loop-counter")
	randomNum  = expvar.NewInt("expvar.example.random-number")
	theHourStr = expvar.NewString("expvar.example.current-hour")
)

func init() {
	for _, a := range animalTypes {
		animals.AddFloat(a, 0.0)
	}

	loopCount.Set(0.0)
	randomNum.Set(0)
	theHourStr.Set("")

	expvar.Publish("expvar.example.wave", waveData)
	expvar.Publish("expvar.example.http.response.times", simulatedHttp)
}

func main() {
	// process command line
	flag.Parse()

	if _, err := strconv.Atoi(*argPort); err != nil {
		panic("Invalid port: " + *argPort)
	}

	// start up the listener
	fmt.Println("Go Expvar Example. Will listen on port", *argPort, "...")
	sock, err := net.Listen("tcp", ":"+*argPort)
	if err != nil {
		panic("Cannot start listener. err=" + err.Error())
	}

	go func() {
		if err := http.Serve(sock, nil); err != nil {
			fmt.Println("HTTP Server stopped.", err.Error())
		}
	}()

	// periodically generate some random metric data every few seconds
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		for _ = range ticker.C {
			// loop counter
			loopCount.Add(1.0)

			// pick an animal and add to its count
			pickAnimal := rand.Intn(len(animalTypes))
			animals.AddFloat(animalTypes[pickAnimal], 1.0)

			// generate a random number
			randomNum.Set(rand.Int63n(100))

			// report what the current hour is local time
			now := time.Now().Local()
			theHourStr.Set(fmt.Sprintf("%4d-%0d-%0d-%0d", now.Year(), now.Month(), now.Day(), now.Hour()))

			// fill in the sin wave from right to left
			amplitude := 10.0
			frequency := 0.025
			t, _ := strconv.ParseFloat(loopCount.String(), 64)
			copy(waveData.data, waveData.data[1:])
			waveData.data[len(waveData.data)-1] = math.Sin(frequency*t) * amplitude

			// create a simulated request and its response time
			simulateHttpRequest()
		}
	}()

	// wait forever, or at least until we are told to exit
	waitForTermination()
}

func waitForTermination() {
	// Channel that is notified when we are done and should exit
	var doneChan = make(chan bool)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			fmt.Println("Termination Signal Received")
			doneChan <- true
		}
	}()

	<-doneChan
}

func simulateHttpRequest() {
	methods := []string{"GET", "POST"}
	paths := []string{
		"/index.html",
		"/store/browse.jsp?product=123",
		"/store/buy.jsp#cart",
		"/admin/query-db",
	}

	// simulate an http request
	method := methods[rand.Intn(len(methods))]
	path := paths[rand.Intn(len(paths))]
	responseTime := rand.Float64() * 10.0 // simulate respond times between 0 and 10 seconds

	// Publish this example metric in two ways:
	// 1. Within a map-in-a-map
	// 2. As a flat metric

	// Map-in-a-map
	methodMap, ok := simulatedHttp[method]
	if !ok {
		methodMap = make(map[string]float64, 0)
		simulatedHttp[method] = methodMap
	}
	methodMap[path] = responseTime

	// Flat
	flatMetricName := fmt.Sprintf("expvar.example.http.response.time.%v-%v", method, path)
	flatMetricVar := expvar.Get(flatMetricName)
	if flatMetricVar == nil {
		flatMetricVar = expvar.NewFloat(flatMetricName)
	}
	flatMetricVar.(*expvar.Float).Set(responseTime)
}
