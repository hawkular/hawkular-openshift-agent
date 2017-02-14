#!/bin/python

from prometheus_client import start_http_server, Gauge
import random
import time
import sys

if __name__ == '__main__':
    print "Hawkular OpenShift Agent Multiple Endpoints Example: Started with arguments: " + str(sys.argv)
    sys.stdout.flush()

    if len(sys.argv) != 4:
        print "Invalid command line arguments. Must be: <low> <high> <port>"
        exit(1)

    # process arguments
    lo = sys.argv[1] # low end of metric value range
    hi = sys.argv[2] # high end of metric value range
    port = sys.argv[3] # port the prometheus server will emit metrics

    metricName = "random_number_" + lo + "_to_" + hi
    metricDesc = "A random number in the range of " + lo + " to " + hi

    metric = Gauge(metricName, metricDesc)

    # Start up the servers to expose the metrics
    start_http_server(int(port))
    print "Hawkular OpenShift Agent Multiple Endpoints Example: Listening to port " + port
    sys.stdout.flush()

    # Generate our random number metric, sleep, repeat
    while True:
        num = random.uniform(float(lo), float(hi))
        metric.set(num)
        sleepTime = random.uniform(1.0,5.0)
        time.sleep(sleepTime)
