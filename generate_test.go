package akamai

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"testing"
)

// TestGenerate tests the Session.Generate method and serves an example of how to design your own custom
// implementations for DoHttpReqFunc and GetCookieFunc. Because this test uses net/http, it will result
// in an invalid cookie being generated.
//
// Users wishing to run this test need to use their own SolarSystems API key by setting the SOLARSYSTEMS_API_KEY
// environment variable when running the test. Users are encouraged to proxy the requests through a request
// sniffer (like Charles Proxy or Fiddler) to see what requests are made.
func TestGenerate(t *testing.T) {
	session := NewSession(os.Getenv("SOLARSYSTEMS_API_KEY"))

	jar, _ := cookiejar.New(nil)
	client := http.Client{
		Jar: jar,
	}

	const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36"

	var doHttpReq DoHttpReqFunc = func(ctx context.Context, op HttpReqOp, requestUrl, requestMethod string, requestBody io.Reader) (statusCode int, responseBody []byte, err error) {
		request, err := http.NewRequestWithContext(ctx, requestMethod, requestUrl, requestBody)
		if err != nil {
			return 0, nil, err
		}

		request.Header.Set("User-Agent", userAgent)
		if op == OpPostPixelPayload {
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

	var getCookie GetCookieFunc = func(u *url.URL, name string) string {
		for _, cookie := range jar.Cookies(u) {
			if cookie.Name == name {
				return cookie.Value
			}
		}

		return ""
	}

	if err := session.Generate(context.Background(), userAgent, "https://www.very.co.uk/", doHttpReq, getCookie, 2); err != nil {
		panic(err)
	}
}
