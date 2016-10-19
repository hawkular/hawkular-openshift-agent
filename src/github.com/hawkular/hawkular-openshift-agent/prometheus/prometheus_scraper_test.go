package prometheus

import (
	"math"
	"os"
	"testing"

	dto "github.com/prometheus/client_model/go"
)

func TestCounter(t *testing.T) {
	mf := scrapeTextFile(t, "testdata/prometheus-counter.txt")

	name := "http_requests_total"

	assertEqualsI(t, 1, len(mf))
	assertEqualsS(t, name, mf[name].GetName())
	assertEqualsS(t, "Total number of HTTP requests made.", mf[name].GetHelp())
	assertEqualsS(t, "COUNTER", mf[name].GetType().String())
	assertEqualsI(t, 5, len(mf[name].GetMetric()))
	assertEqualsF(t, 162030, mf[name].GetMetric()[0].GetCounter().GetValue())
	assertEqualsI(t, 3, len(mf[name].GetMetric()[0].GetLabel()))

	for _, label := range mf[name].GetMetric()[0].GetLabel() {
		switch label.GetName() {
		case "code":
			{
				assertEqualsS(t, "200", label.GetValue())
			}
		case "handler":
			{
				assertEqualsS(t, "prometheus", label.GetValue())
			}
		case "method":
			{
				assertEqualsS(t, "get", label.GetValue())
			}
		default:
			{
				t.Fatalf("Unexpected label: %v", label.GetName())
			}
		}
	}

	// test NaN, -Inf, and +Inf
	assertEqualsF(t, math.NaN(), mf[name].GetMetric()[2].GetCounter().GetValue())
	assertEqualsF(t, math.Inf(+1), mf[name].GetMetric()[3].GetCounter().GetValue())
	assertEqualsF(t, math.Inf(-1), mf[name].GetMetric()[4].GetCounter().GetValue())
}

func TestGauge(t *testing.T) {
	mf := scrapeTextFile(t, "testdata/prometheus-gauge.txt")

	name := "go_memstats_alloc_bytes"

	assertEqualsI(t, 1, len(mf))
	assertEqualsS(t, name, mf[name].GetName())
	assertEqualsS(t, "", mf[name].GetHelp())
	assertEqualsS(t, "GAUGE", mf[name].GetType().String())
	assertEqualsI(t, 1, len(mf[name].GetMetric()))
	assertEqualsF(t, 4.14422136e+08, mf[name].GetMetric()[0].GetGauge().GetValue())
	assertEqualsI(t, 0, len(mf[name].GetMetric()[0].GetLabel()))
}

func TestSummary(t *testing.T) {
	mf := scrapeTextFile(t, "testdata/prometheus-summary.txt")

	name := "prometheus_local_storage_maintain_series_duration_milliseconds"

	assertEqualsI(t, 1, len(mf))
	assertEqualsS(t, name, mf[name].GetName())
	assertEqualsS(t, "The duration (in milliseconds) it took to perform maintenance on a series.", mf[name].GetHelp())
	assertEqualsS(t, "SUMMARY", mf[name].GetType().String())
	assertEqualsI(t, 2, len(mf[name].GetMetric()))

	// the metrics should appear in order as they appear in the text. First is "memory", second is "disk"
	assertEqualsI(t, 1, len(mf[name].GetMetric()[0].GetLabel()))
	assertEqualsS(t, "location", mf[name].GetMetric()[0].GetLabel()[0].GetName())
	assertEqualsS(t, "memory", mf[name].GetMetric()[0].GetLabel()[0].GetValue())

	assertEqualsI(t, 1, len(mf[name].GetMetric()[1].GetLabel()))
	assertEqualsS(t, "location", mf[name].GetMetric()[1].GetLabel()[0].GetName())
	assertEqualsS(t, "disk", mf[name].GetMetric()[1].GetLabel()[0].GetValue())

	assertEqualsI(t, 12345, int(mf[name].GetMetric()[1].GetSummary().GetSampleCount()))
	assertEqualsF(t, 1.5, mf[name].GetMetric()[1].GetSummary().GetSampleSum())
	assertEqualsF(t, 00.50, mf[name].GetMetric()[1].GetSummary().GetQuantile()[0].GetQuantile())
	assertEqualsF(t, 40.00, mf[name].GetMetric()[1].GetSummary().GetQuantile()[0].GetValue())
	assertEqualsF(t, 00.90, mf[name].GetMetric()[1].GetSummary().GetQuantile()[1].GetQuantile())
	assertEqualsF(t, 50.00, mf[name].GetMetric()[1].GetSummary().GetQuantile()[1].GetValue())
	assertEqualsF(t, 00.99, mf[name].GetMetric()[1].GetSummary().GetQuantile()[2].GetQuantile())
	assertEqualsF(t, 60.00, mf[name].GetMetric()[1].GetSummary().GetQuantile()[2].GetValue())
}

func TestHistogram(t *testing.T) {
	mf := scrapeTextFile(t, "testdata/prometheus-histogram.txt")

	name := "http_request_duration_seconds"

	assertEqualsI(t, 1, len(mf))
	assertEqualsS(t, name, mf[name].GetName())
	assertEqualsS(t, "A histogram of the request duration.", mf[name].GetHelp())
	assertEqualsS(t, "HISTOGRAM", mf[name].GetType().String())
	assertEqualsI(t, 1, len(mf[name].GetMetric()))

	assertEqualsI(t, 1, len(mf[name].GetMetric()[0].GetLabel()))
	assertEqualsS(t, "mylabel", mf[name].GetMetric()[0].GetLabel()[0].GetName())
	assertEqualsS(t, "wotgorilla?", mf[name].GetMetric()[0].GetLabel()[0].GetValue())

	assertEqualsI(t, 144320, int(mf[name].GetMetric()[0].GetHistogram().GetSampleCount()))
	assertEqualsF(t, 53423, mf[name].GetMetric()[0].GetHistogram().GetSampleSum())

	assertEqualsF(t, 0.05, mf[name].GetMetric()[0].GetHistogram().GetBucket()[0].GetUpperBound())
	assertEqualsI(t, 24054, int(mf[name].GetMetric()[0].GetHistogram().GetBucket()[0].GetCumulativeCount()))
	assertEqualsF(t, 0.1, mf[name].GetMetric()[0].GetHistogram().GetBucket()[1].GetUpperBound())
	assertEqualsI(t, 33444, int(mf[name].GetMetric()[0].GetHistogram().GetBucket()[1].GetCumulativeCount()))
	assertEqualsF(t, 0.2, mf[name].GetMetric()[0].GetHistogram().GetBucket()[2].GetUpperBound())
	assertEqualsI(t, 100392, int(mf[name].GetMetric()[0].GetHistogram().GetBucket()[2].GetCumulativeCount()))
	assertEqualsF(t, 0.5, mf[name].GetMetric()[0].GetHistogram().GetBucket()[3].GetUpperBound())
	assertEqualsI(t, 129389, int(mf[name].GetMetric()[0].GetHistogram().GetBucket()[3].GetCumulativeCount()))
	assertEqualsF(t, 1.0, mf[name].GetMetric()[0].GetHistogram().GetBucket()[4].GetUpperBound())
	assertEqualsI(t, 133988, int(mf[name].GetMetric()[0].GetHistogram().GetBucket()[4].GetCumulativeCount()))
	assertEqualsF(t, math.Inf(+1), mf[name].GetMetric()[0].GetHistogram().GetBucket()[5].GetUpperBound())
	assertEqualsI(t, 144320, int(mf[name].GetMetric()[0].GetHistogram().GetBucket()[5].GetCumulativeCount()))
}

func TestBigText(t *testing.T) {
	// there should be 72 metric families with a total of 127 individual metrics total
	mf := scrapeTextFile(t, "testdata/prometheus.txt")
	assertEqualsI(t, 72, len(mf))
	counter := 0
	for _, v := range mf {
		counter += len(v.GetMetric())
	}
	assertEqualsI(t, 127, counter)
}

func TestBigBinary(t *testing.T) {
	// there should be 71 metric families with a total of 126 individual metrics total
	mf := scrapeBinaryFile(t, "testdata/prometheus.data")
	assertEqualsI(t, 71, len(mf))
	counter := 0
	for _, v := range mf {
		counter += len(v.GetMetric())
	}
	assertEqualsI(t, 126, counter)
}

func scrapeTextFile(t *testing.T, filename string) (mf map[string]*dto.MetricFamily) {
	file, err := os.Open(filename)
	if err != nil {
		t.Fatalf("Failed to open text file: err=%v", err)
		return
	}
	defer file.Close()
	mf, err = ScrapeText(file)
	if err != nil {
		t.Fatalf("Failed to scrape text file: err=%v", err)
		return
	}
	return mf
}

func scrapeBinaryFile(t *testing.T, filename string) (mf map[string]*dto.MetricFamily) {
	file, err := os.Open(filename)
	if err != nil {
		t.Fatalf("Failed to open binary file: err=%v", err)
		return
	}
	defer file.Close()
	mf, err = ScrapeBinary(file)
	if err != nil {
		t.Fatalf("Failed to scrape binary file: err=%v", err)
		return
	}
	return mf
}

func assertEqualsF(t *testing.T, expected float64, actual float64) {
	//t.Fatalf("%v <-> %v\n", expected, actual)

	if (math.IsNaN(expected) && !math.IsNaN(actual)) ||
		(math.IsNaN(actual) && !math.IsNaN(expected)) {
		t.Fatalf("Expected [%v] but got [%v]", expected, actual)
	}

	if (math.IsInf(expected, +1) && !math.IsInf(actual, +1)) ||
		(math.IsInf(actual, +1) && !math.IsInf(expected, +1)) {
		t.Fatalf("Expected [%v] but got [%v]", expected, actual)
	}

	if (math.IsInf(expected, -1) && !math.IsInf(actual, -1)) ||
		(math.IsInf(actual, -1) && !math.IsInf(expected, -1)) {
		t.Fatalf("Expected [%v] but got [%v]", expected, actual)
	}

	diff := float64(0.001)
	if ((expected - actual) > diff) || ((actual - expected) > diff) {
		t.Fatalf("Expected [%v] but got [%v]", expected, actual)
	}
}

func assertEqualsI(t *testing.T, expected int, actual int) {
	if expected != actual {
		t.Fatalf("Expected [%v] but got [%v]", expected, actual)
	}
}

func assertEqualsS(t *testing.T, expected string, actual string) {
	if expected != actual {
		t.Fatalf("Expected [%v] but got [%v]", expected, actual)
	}
}
