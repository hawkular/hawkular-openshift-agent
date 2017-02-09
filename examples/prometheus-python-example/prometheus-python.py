#!/bin/python
from prometheus_client import start_http_server, Gauge, Counter
import random
import time
import sys

# Create some test metrics
TEST_SLEEP_GAUGE = Gauge('test_last_sleep_duration', 'The duration of the last sleep in seconds')
TEST_METHOD_COUNTER = Counter('test_method_invocations_total', 'Number of times methods are called', ['method_name'])
TEST_MEAL_COUNTER = Counter('test_meals_eaten_total', 'Number of times a meal was eaten', ['food', 'drink'])

def do_random_sleep():
    TEST_METHOD_COUNTER.labels('do_random_sleep').inc()
    t = random.uniform(0.1,3.0)
    TEST_SLEEP_GAUGE.set(t)
    time.sleep(t)

def eat():
    TEST_METHOD_COUNTER.labels('eat').inc()
    food = ['Apple','Banana','Hamburger']
    drink = ['Water','Lemonade','Beer']
    pick_food = random.randint(0, len(food)-1)
    pick_drink = random.randint(0, len(drink)-1)
    TEST_MEAL_COUNTER.labels(food[pick_food], drink[pick_drink]).inc()
    #print "Hawkular OpenShift Agent Prometheus Example: Eating " + food[pick_food] + " with " + drink[pick_drink]

if __name__ == '__main__':
    print "Hawkular OpenShift Agent Prometheus Example: Started..."
    sys.stdout.flush()

    # Start up the server to expose the metrics.
    start_http_server(8181)
    print "Hawkular OpenShift Agent Prometheus Example: Listening to port 8181..."
    sys.stdout.flush()

    # Go to sleep and when I get up see if I want to eat (75% chance I'm hungry)
    while True:
        do_random_sleep()
        should_i_eat = (random.randint(1,100) > 25)
        if should_i_eat:
            eat()
