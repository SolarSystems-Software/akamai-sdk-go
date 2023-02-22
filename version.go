package akamai

import "regexp"

// Version represents an Akamai Bot Manager web SDK version.
type Version string

const (
	// Version17 is Akamai 1.7.
	Version17 Version = "1.7"

	// Version175 is Akamai 1.75.
	Version175 Version = "1.75"

	// Version2 is Akamai 2.0.
	Version2 Version = "2"
)

var (
	version175expr = regexp.MustCompile(`^var _acxj`)
	version2expr   = regexp.MustCompile(`^\(function`)
)

// GetSdkVersion gets the Akamai Bot Manager SDK version from the given JavaScript code src.
func GetSdkVersion(src []byte) Version {
	if version175expr.Match(src) {
		return Version175
	} else if version2expr.Match(src) {
		return Version2
	} else {
		return Version17
	}
}
