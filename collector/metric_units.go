package collector

import (
	"fmt"
	"strings"
)

type MetricUnits struct {
	Symbol string
	Custom bool
}

const customMetricUnitsPrefix = "custom:"

const unknown = "Unknown"

type standardMetricUnits []MetricUnits

// standardMetricUnitsList is a list of standard metric units
// See https://en.wikipedia.org/wiki/International_System_of_Units and http://metrics20.org/spec/
var standardMetricUnitsList = standardMetricUnits{
	// absolute sizes in bytes
	{Symbol: "B"},
	{Symbol: "kB"},
	{Symbol: "MB"},
	{Symbol: "GB"},
	{Symbol: "TB"},
	{Symbol: "PB"},
	{Symbol: "KiB"},
	{Symbol: "MiB"},
	{Symbol: "GiB"},
	{Symbol: "TiB"},
	{Symbol: "PiB"},

	// absolute sizes in bits
	{Symbol: "b"},
	{Symbol: "kb"},
	{Symbol: "Mb"},
	{Symbol: "Gb"},
	{Symbol: "Tb"},
	{Symbol: "Pb"},
	{Symbol: "Kib"},
	{Symbol: "Mib"},
	{Symbol: "Gib"},
	{Symbol: "Tib"},
	{Symbol: "Pib"},

	// relative time
	{Symbol: "jiff"},
	{Symbol: "ns"},
	{Symbol: "us"},
	{Symbol: "ms"},
	{Symbol: "s"},
	{Symbol: "M"},
	{Symbol: "h"},
	{Symbol: "d"},
	{Symbol: "w"},

	// frequency
	{Symbol: "Hz"},
	{Symbol: "kHz"},
	{Symbol: "MHz"},
	{Symbol: "GHz"},

	// temperature
	{Symbol: "C"},
	{Symbol: "F"},
	{Symbol: "K"},

	// current
	{Symbol: "uA"},
	{Symbol: "mA"},
	{Symbol: "A"},
	{Symbol: "kA"},
	{Symbol: "MA"},
	{Symbol: "GA"},

	// voltage
	{Symbol: "uV"},
	{Symbol: "mV"},
	{Symbol: "V"},
	{Symbol: "kV"},
	{Symbol: "MV"},
	{Symbol: "GV"},

	// percentage
	{Symbol: "%"},

	// none (no metric units are applicable)
	{Symbol: ""},
}

// GetMetricUnits will check to see if the given string is a valid units identifier.
// If it is valid, this returns the units string as a MetricUnits.
// If it is not a valid units identifier, an error is returned.
// If it is a custom units identifier, it is returned minus the custom units prefix.
func GetMetricUnits(u string) (MetricUnits, error) {
	if len(u) > len(customMetricUnitsPrefix) && strings.HasPrefix(u, customMetricUnitsPrefix) {
		mu := MetricUnits{
			Symbol: strings.TrimPrefix(u, customMetricUnitsPrefix),
			Custom: true,
		}
		return mu, nil
	}
	for _, x := range standardMetricUnitsList {
		if x.Symbol == u {
			return x, nil
		}
	}
	return MetricUnits{Symbol: unknown}, fmt.Errorf("invalid metric units: %v", u)
}
