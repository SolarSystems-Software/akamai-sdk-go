# akamai-sdk-go
API SDK for usage with the SolarSystems Akamai API.

## Example usage
This example uses the `Session.Generate` function, which automatically generates cookies for you.
It requires function implementations to execute HTTP requests on your behalf (using your own TLS client)
and a function to get a cookie by its name.

```go
package main

import (
	"context"
	"github.com/SolarSystems-Software/akamai-sdk-go"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
)

func main() {
	// Create an API session; this can be re-used across multiple tasks
	session := akamai.NewSession(os.Getenv("SOLARSYSTEMS_API_KEY"))

	// Your client's cookie jar
	jar, _ := cookiejar.New(nil)
	client := http.Client{
		Jar: jar,
	}

	// The user agent to use
	const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36"

	// The function that executes HTTP requests with your HTTP client
	var doHttpReq DoHttpReqFunc = func(ctx context.Context, op HttpReqOp, requestUrl, requestMethod string, requestBody io.Reader) (statusCode int, responseBody []byte, err error) {
		request, err := http.NewRequestWithContext(ctx, requestMethod, requestUrl, requestBody)
		if err != nil {
			return 0, nil, err
		}

		/*
		In a real use case, to implement header ordering and header values per request, you'd need
		a switch statement like this:
		
		switch op {
		case akamai.OpGetPage:
		case akamai.OpPostSensorData:
		...
		}
		*/

		request.Header.Set("User-Agent", userAgent)
		if op == akamai.OpPostPixelPayload {
			// This header is required for solving the pixel challenge.
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		response, err := client.Do(request)
		if err != nil {
			return 0, nil, err
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return 0, nil, err
		}

		return response.StatusCode, body, nil
	}

	// The function that obtains a cookie value by its URL and name
	var getCookie GetCookieFunc = func(u *url.URL, name string) string {
		for _, cookie := range jar.Cookies(u) {
			if cookie.Name == name {
				return cookie.Value
			}
		}

		return ""
	}

	// Generate cookies. This will solve the pixel challenge and generate _abck.
	// This will also handle all possible outcomes automatically (like dynamic scripts).
	//
	// In production, you should use a custom context with a select statement if you wish to have a timeout.
	if err := session.Generate(context.Background(), userAgent, "https://www.very.co.uk/", doHttpReq, getCookie, 2); err != nil {
		panic(err)
	}
}
```