package akamai

import (
	"strconv"
	"strings"
)

// IsCookieValid reports if the given `_abck` cookie value is valid using Akamai Bot Manager's client-side
// stop signal logic with the provided request count. If `true`, the client SHOULD stop posting sensor data.
// Any further posting will still result in a valid cookie, but is redundant.
//
// Stop signal is client-side logic Akamai Bot Manager uses to inform a client that the received cookie
// is valid and that any further posting is redundant.
//
// Not all applications have stop signal enabled. In this case, the client keeps posting whenever an event
// is fired. In such case, there is no way to tell if a cookie is truly valid without using it in a request
// to a protected endpoint. Sensor data obtained from the SolarSystems API typically requires one POST
// request to obtain a valid cookie, or two if the application uses challenges.
func IsCookieValid(value string, requestCount int) bool {
	parts := strings.Split(value, "~")
	if len(parts) < 2 {
		return false
	}

	requestThreshold, err := strconv.Atoi(parts[1])
	if err != nil {
		requestThreshold = -1
	}
	return requestThreshold != -1 && requestCount >= requestThreshold
}
