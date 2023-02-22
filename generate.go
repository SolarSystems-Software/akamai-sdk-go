package akamai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// GenerateRequest is the API generation request schema.
type GenerateRequest struct {
	// UserAgent is the user agent to use when generating sensor data.
	//
	// Current restrictions apply to this preference. The user agent
	// must be a Google Chrome v109 or v110 user agent.
	// Callers can use any platform they like, however it is highly
	// recommended to use Windows.
	UserAgent string `json:"userAgent"`

	// Version is the Akamai version to use.
	Version Version `json:"version"`

	// PageURL is the URL of the page to generate sensor data for.
	PageURL string `json:"pageUrl"`

	// Abck is the current `_abck` cookie.
	Abck string `json:"_abck"`

	// BmSz is the current `bm_sz` cookie.
	//
	// This is optional for every version excluding `2`.
	BmSz string `json:"bm_sz,omitempty"`

	// ScriptValues are the dynamic script values.
	//
	// This should be empty for static scripts.
	// See GetDynamicScriptValues for more information.
	ScriptValues string `json:"scriptValues,omitempty"`
}

// GenerateResponse is the API generation response schema.
type GenerateResponse struct {
	// Payload is the sensor data.
	Payload string `json:"payload"`
}

// GenerateSensorData generates sensor data to use to post to an Akamai Bot Manager
// script endpoint to obtain cookies.
//
// It is recommended to use Generate instead as it includes stop signal handling and
// is generally easier to use.
//
// When sending POST requests to generate an `_abck` cookie with the generated sensor data,
// callers SHOULD NOT encode the data as JSON. Akamai Bot Manager sends the payload as JSON,
// but does not properly encode the data as JSON. Because of this, the request body should be
// created as such: `{"sensor_data":"` + <generated sensor data> + `"}`.
// Callers using Generate do not need to worry about this requirement as Generate
// handles this automatically.
func (session Session) GenerateSensorData(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://akamai.publicapis.solarsystems.software/v1/sensor/generate", bytes.NewBuffer(encoded))
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "SolarSystems akamai-sdk-go")
	request.Header.Set("x-api-key", session.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := session.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusCreated {
		return nil, ApiOperationError{
			StatusCode: response.StatusCode,
			Message:    GetMessageFromErrorResponse(body),
		}
	}

	var resp GenerateResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

var (
	// ErrInvalidPageURL is an error caused by Session.Generate if the provided page URL is not
	// a valid or absolute URL. An absolute URL must contain a scheme and host.
	ErrInvalidPageURL = errors.New("akamai-sdk-go: invalid page URL")
)

// Generate generates a set of cookies (_abck, bm_sz, ak_bmsc and possibly others) to use in an HTTP
// request to an API endpoint protected by Akamai Bot Manager. This method handles all possible scenarios
// and outcomes of generation for both the Akamai Bot Manager web SDK ("sensor data") and the pixel challenge
// (if it is present and not solved already). Callers simply have to provide an implementation of DoHttpReqFunc
// and GetCookieFunc; see their documentation for instructions on how to make custom implementations.
//
// Generate makes an HTTP GET request to the given page URL and obtains the required variables from the
// HTML document (pixel challenge script location and web SDK script location). It then makes HTTP requests
// to both scripts, and sends POST requests containing payloads to generate cookies. The pixel challenge
// and sensor data generation happens concurrently, meaning the order of the sent requests may not
// always be the same. Implementations should use a mutex if they need concurrency safety in their implementation
// of DoHttpReqFunc or GetCookieFunc as both functions can be called by multiple goroutines.
//
// Sensor data generation sends a maximum of maxTries requests, after which it gives up. Generation will stop sooner
// if the website uses the stop signal feature; see IsCookieValid for more information.
// Websites typically require one POST request with sensor data from the SolarSystems API to generate a valid _abck
// cookie. Websites with challenges require two. Setting maxTries to two is a reasonable choice.
//
// Generate blocks until solving the pixel challenge and generating an _abck is complete. It is safe for usage
// by multiple goroutines.
//
// Generate panics if doHttpReq or getCookie is nil. pageUrl must also be an absolute URL, and maxTries must be
// a positive, non-zero integer.
func (session Session) Generate(
	ctx context.Context,
	userAgent,
	pageUrl string,
	doHttpReq DoHttpReqFunc,
	getCookie GetCookieFunc,
	maxTries int,
) error {
	if doHttpReq == nil {
		panic("akamai-sdk-go: nil DoHttpReqFunc passed to Generate")
	}
	if getCookie == nil {
		panic("akamai-sdk-go: nil GetCookieFunc passed to Generate")
	}
	if maxTries <= 0 {
		panic("akamai-sdk-go: maxTries <= 0")
	}

	// We don't need the parsed URL until later, but we parse it now to ensure it's valid and absolute.
	// This will avoid wasting a request if it's invalid.
	u, err := url.Parse(pageUrl)
	if err != nil {
		return err
	}
	if !u.IsAbs() {
		return ErrInvalidPageURL
	}

	// GET pageUrl
	statusCode, pageBody, err := doHttpReq(ctx, OpGetPage, pageUrl, http.MethodGet, nil)
	if err == nil && statusCode != http.StatusOK {
		err = BadStatusCodeError{StatusCode: statusCode}
	}
	if err != nil {
		return errors.Join(HttpOpError{Op: OpGetPage}, err)
	}

	// wg is the WaitGroup for all worker goroutines.
	var wg sync.WaitGroup
	wg.Add(2)
	// errs are the errors reported by the workers.
	var errs []error
	var mu sync.Mutex
	// addError appends to errs. It is safe for usage by multiple goroutines.
	addError := func(err error) {
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
	}

	// Solve pixel challenge
	go func() {
		defer wg.Done()

		// Get the script's URL and the URL to post the payload to
		ok, scriptUrl, postUrl := GetPixelChallengeScriptURL(pageBody)
		if !ok {
			// Pixel challenge is not present on this page.
			return
		}

		// Get the HTML variable
		htmlVar, err := GetPixelChallengeHtmlVar(pageBody)
		if err != nil {
			addError(err)
			return
		}

		// GET request to pixel script
		statusCode, scriptBody, err := doHttpReq(ctx, OpGetPixelChallengeScript, scriptUrl, http.MethodGet, nil)
		if err == nil && statusCode != http.StatusOK {
			if statusCode == http.StatusNotFound {
				// Pixel challenge script returns 404 when the challenge is already solved.
				return
			}

			err = BadStatusCodeError{StatusCode: statusCode}
		}
		if err != nil {
			addError(err)
			return
		}

		// Get dynamic script variable
		scriptVar, err := GetPixelChallengeScriptVar(scriptBody)
		if err != nil {
			addError(err)
			return
		}

		// Generate payload
		response, err := session.GeneratePixelPayload(ctx, &PixelSolveRequest{
			UserAgent: userAgent,
			HtmlVar:   htmlVar,
			ScriptVar: scriptVar,
		})
		if err != nil {
			addError(err)
			return
		}

		// POST payload
		if _, _, err = doHttpReq(
			ctx,
			OpPostPixelPayload,
			postUrl,
			http.MethodPost,
			bytes.NewBufferString(response.Payload),
		); err != nil {
			addError(err)
			return
		}
	}()

	// Generate _abck
	go func() {
		defer wg.Done()

		// Get script path
		scriptPath := GetScriptPath(pageBody)
		if scriptPath == "" {
			// If there's no script path on the page then we skip generating.
			return
		}
		// Construct script URL -- scriptPath will always begin with a /
		scriptUrl := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, scriptPath)

		// GET request to script
		statusCode, scriptBody, err := doHttpReq(ctx, OpGetSdkScript, scriptUrl, http.MethodGet, nil)
		if err == nil && statusCode != http.StatusOK {
			err = BadStatusCodeError{StatusCode: statusCode}
		}
		if err != nil {
			addError(err)
			return
		}

		// Get SDK version
		version := GetSdkVersion(scriptBody)

		// Get dynamic script values.
		// This will be a blank string if the script is static.
		var scriptValues string
		if version == Version2 && !IsScriptStatic(scriptBody) {
			response, err := session.GetDynamicScriptValues(ctx, scriptBody)
			if err != nil {
				addError(err)
				return
			}

			if response.Success {
				scriptValues = response.ScriptValues
			}
		}

		// Generate and post sensor data
		for i := 0; i < maxTries; i++ {
			request := GenerateRequest{
				UserAgent:    userAgent,
				Version:      version,
				PageURL:      pageUrl,
				Abck:         getCookie(u, "_abck"),
				ScriptValues: scriptValues,
			}
			if version == Version2 {
				request.BmSz = getCookie(u, "bm_sz")
			}

			response, err := session.GenerateSensorData(ctx, &request)
			if err != nil {
				addError(err)
				return
			}

			if _, _, err = doHttpReq(
				ctx,
				OpPostSensorData,
				scriptUrl,
				http.MethodPost,
				bytes.NewBufferString(fmt.Sprintf(`{"sensor_data":"%s"}`, response.Payload)),
			); err != nil {
				addError(err)
				return
			}

			if IsCookieValid(getCookie(u, "_abck"), i) {
				break
			}
		}
	}()

	wg.Wait()
	return errors.Join(errs...)
}
