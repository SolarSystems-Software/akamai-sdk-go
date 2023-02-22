package akamai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SolarSystems-Software/akamai-sdk-go/internal"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	pixelHtmlExpr = regexp.MustCompile(`bazadebezolkohpepadr="(\d+)"`)

	ErrPixelHtmlVarNotFound = errors.New("akamai-sdk-go: pixel HTML var not found")
)

// GetPixelChallengeHtmlVar gets the required pixel challenge variable from the given HTML code src.
//
// The error returned is non-nil if the value was not found. In this case, the returned error
// is ErrPixelHtmlVarNotFound. There may be multiple errors; callers can use errors.Unwrap
// to get all the errors.
func GetPixelChallengeHtmlVar(src []byte) (int, error) {
	matches := pixelHtmlExpr.FindSubmatch(src)
	if len(matches) < 2 {
		return 0, ErrPixelHtmlVarNotFound
	}

	if v, err := strconv.Atoi(string(matches[1])); err == nil {
		return v, nil
	} else {
		return 0, errors.Join(ErrPixelHtmlVarNotFound, err)
	}
}

var pixelScriptUrlExpr = regexp.MustCompile(`(?i)src="(https?://.+/akam/\d+/\w+)"`)

// GetPixelChallengeScriptURL gets the script URL of the pixel challenge script and the URL
// to post a generated payload to from the given HTML code src.
//
// ok is true if the URL was found. Callers should treat ok == false as an error.
// See GetPixelChallengeHtmlVar for more information.
func GetPixelChallengeScriptURL(src []byte) (ok bool, scriptUrl, postUrl string) {
	matches := pixelScriptUrlExpr.FindSubmatch(src)
	if len(matches) < 2 {
		return
	}

	scriptUrl = string(matches[1])

	// Create postUrl
	parts := strings.Split(scriptUrl, "/")
	parts[len(parts)-1] = "pixel_" + parts[len(parts)-1]
	postUrl = strings.Join(parts, "/")

	ok = true
	return
}

var (
	pixelScriptVarExpr         = regexp.MustCompile(`g=_\[(\d+)]`)
	pixelScriptStringArrayExpr = regexp.MustCompile(`(?i)var _=\[(.+)];`)
	pixelScriptStringsExpr     = regexp.MustCompile(`(?i)"([^",]*)"`)

	ErrPixelScriptVarNotFound = errors.New("akamai-sdk-go: script var not found")
)

// GetPixelChallengeScriptVar gets the dynamic pixel challenge variable from the given JavaScript code src.
// The JavaScript code should be the pixel script.
//
// err is nil if the value of the variable was found. In all other cases, ErrPixelScriptVarNotFound is the
// returned error, which contains another error explaining in detail why the call resulted in an error.
// Callers can use errors.Unwrap to get the more detailed error.
func GetPixelChallengeScriptVar(src []byte) (string, error) {
	// Find array index
	index := pixelScriptVarExpr.FindSubmatch(src)
	if length := len(index); length != 2 {
		return "", errors.Join(ErrPixelScriptVarNotFound, fmt.Errorf("len(index) expected 2, got: %d", length))
	}
	stringIndex, err := strconv.Atoi(string(index[1]))
	if err != nil {
		return "", errors.Join(ErrPixelScriptVarNotFound, err)
	}

	// Find array with encoded strings
	arrayDeclaration := pixelScriptStringArrayExpr.FindSubmatch(src)
	if length := len(arrayDeclaration); length < 2 {
		return "", errors.Join(
			ErrPixelScriptVarNotFound,
			fmt.Errorf("len(arrayDeclaration) expected 2, got: %d", length),
		)
	}

	// The raw strings
	rawStrings := pixelScriptStringsExpr.FindAllSubmatch(arrayDeclaration[1], -1)
	// bounds check to prevent a possible panic
	if stringIndex >= len(rawStrings) {
		return "", errors.Join(
			ErrPixelScriptVarNotFound,
			fmt.Errorf("string index out of range: %d >= %d", stringIndex, len(rawStrings)),
		)
	}

	if length := len(rawStrings[stringIndex]); length != 2 {
		return "", errors.Join(
			ErrPixelScriptVarNotFound,
			fmt.Errorf("len(rawStrings[stringIndex]) expected 2, got: %d", length),
		)
	}

	if v, err := internal.FromHexString(string(rawStrings[stringIndex][1])); err == nil {
		return v, nil
	} else {
		return "", errors.Join(ErrPixelScriptVarNotFound, err)
	}
}

// PixelSolveRequest is the API pixel challenge request schema.
type PixelSolveRequest struct {
	UserAgent string `json:"userAgent"`
	HtmlVar   int    `json:"htmlVar"`
	ScriptVar string `json:"scriptVar"`
}

// PixelSolveResponse is the API pixel challenge response schema.
type PixelSolveResponse struct {
	Payload string `json:"payload"`
}

// GeneratePixelPayload generates a payload to use to solve the pixel challenge with the given variables
// obtained from GetPixelChallengeHtmlVar and GetPixelChallengeScriptVar.
//
// Callers should use postUrl obtained from GetPixelChallengeScriptURL to send a POST request with
// the generated payload to.
// When sending the POST request to solve the challenge, callers MUST set the Content-Type HTTP
// request header to "application/x-www-form-urlencoded" (without quotes).
// Failure to do so will result in the challenge being treated as invalid by Akamai Bot Manager.
//
// Callers should prefer Generate over this method as Generate will handle generation
// logic and processing automatically. This method is intended for callers who wish to interact with
// the API directly.
func (session Session) GeneratePixelPayload(ctx context.Context, req *PixelSolveRequest) (*PixelSolveResponse, error) {
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://akamai.publicapis.solarsystems.software/v1/pixel/generate",
		bytes.NewBuffer(encoded),
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "SolarSystems akamai-sdk-go")
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

	var resp PixelSolveResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
