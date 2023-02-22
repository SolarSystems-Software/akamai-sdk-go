package akamai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// HttpReqOp represents the cause of a request being executed with a DoHttpReqFunc.
type HttpReqOp byte

func (op HttpReqOp) String() string {
	switch op {
	case OpGetPage:
		return "OpGetPage"
	case OpGetSdkScript:
		return "OpGetSdkScript"
	case OpPostSensorData:
		return "OpPostSensorData"
	case OpGetPixelChallengeScript:
		return "OpGetPixelChallengeScript"
	case OpPostPixelPayload:
		return "OpPostPixelPayload"
	default:
		return ""
	}
}

const (
	// OpGetPage sends a GET request to a document.
	OpGetPage HttpReqOp = iota

	// OpGetSdkScript sends a GET request to the Akamai Bot Manager web SDK.
	OpGetSdkScript

	// OpPostSensorData sends a POST request with sensor data to the Akamai Bot Manager web SDK endpoint.
	OpPostSensorData

	// OpGetPixelChallengeScript sends a GET request to the pixel challenge script (if it exists).
	OpGetPixelChallengeScript

	// OpPostPixelPayload sends a POST request with a payload to solve the pixel challenge.
	// This operation only occurs if the script exists AND the challenge isn't already solved.
	// See Session.Generate for more information.
	OpPostPixelPayload
)

// HttpOpError is a generic HTTP request failure. It contains no information about the actual
// error that occurred; callers should use errors.Unwrap to get a detailed cause.
type HttpOpError struct {
	// Op is the attempted operation that failed.
	Op HttpReqOp
}

func (e HttpOpError) Error() string {
	return fmt.Sprintf("akamai-sdk-go: http req op error for operation type %s", e.Op)
}

// BadStatusCodeError is an error caused by an undesirable HTTP status code.
type BadStatusCodeError struct {
	// StatusCode is the HTTP status code that caused the error.
	StatusCode int
}

func (e BadStatusCodeError) Error() string {
	return fmt.Sprintf("akamai-sdk-go: bad status HTTP %d %s", e.StatusCode, http.StatusText(e.StatusCode))
}

// DoHttpReqFunc makes an HTTP request to the provided request URL with the given request method
// and request body. The implementation should return the status code, response body,
// and an error (if any). Implementations MUST return an empty slice for the response body if
// the body is empty. Returning a nil body if the returned error is nil will cause a panic when
// used in functions like Session.Generate. Implementations should also close the response body
// to prevent resource leaking.
//
// The returned error MUST be nil unless an error occurred executing the HTTP request.
// Implementations MUST NOT return an error due to an undesirable HTTP status code; functions
// like Session.Generate handle this automatically.
//
// Implementations can use the op parameter to differentiate different types of requests.
// This is how implementations should decide on which headers to set and which order to set them in.
//
// Functions of this type are implemented by the caller to allow full control over HTTP requests
// for the caller. The main feature of this is to allow callers to use their own TLS fingerprint
// with a custom HTTP client other than net/http.
//
// Implementations should not keep references to requestBody or the data that it stores.
// The request body may also be nil; implementations should pass the given io.Reader directly
// into the request they create.
//
// Unlike the request body, implementations can keep references of the response body if the
// implementation wishes to keep the response. This is useful in cases where functions
// like Session.Generate makes a request to a document (like a product page) and you want to
// keep the document body without having to send a second request.
type DoHttpReqFunc func(
	ctx context.Context,
	op HttpReqOp,
	requestUrl,
	requestMethod string,
	requestBody io.Reader,
) (
	statusCode int,
	responseBody []byte,
	err error,
)

// GetCookieFunc gets an HTTP cookie's value by its name for the given URL.
//
// Implementations MUST return an empty string if the cookie with the given name doesn't exist.
type GetCookieFunc func(u *url.URL, name string) string
