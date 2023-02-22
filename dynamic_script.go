package akamai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
)

var (
	staticScriptExpr = regexp.MustCompile(`^\(function \w+?`)
)

// IsScriptStatic reports if the given JavaScript code src is a static script.
//
// Akamai Bot Manager currently has three variations of the web SDK:
//
//   - Static scripts. These scripts match the regular expression staticScriptExpr.
//
//   - Dynamic scripts. These scripts do NOT match the regular expression staticScriptExpr.
//
//   - Dynamic scripts (new static variant). These are like the dynamic scripts and have
//     mixed behaviour of both the static and regular dynamic version.
//
// IsScriptStatic returns true if the SDK is the static version, and false for both of
// the dynamic variants.
func IsScriptStatic(src []byte) bool {
	return staticScriptExpr.Match(src)
}

// DynamicScriptResponse is the dynamic script API response schema.
type DynamicScriptResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	ScriptValues string `json:"scriptValues,omitempty"`
}

// GetDynamicScriptValues gets the value required for generation for dynamic scripts from
// the given JavaScript code src.
//
// Callers should validate that src is a dynamic script with IsScriptStatic before
// calling this function. If GetDynamicScriptValues is called with a static script
// (IsScriptStatic(src) == false), then the `success` property of DynamicScriptResponse will
// be false.
func (session Session) GetDynamicScriptValues(ctx context.Context, src []byte) (*DynamicScriptResponse, error) {
	encodedSrc := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(encodedSrc, src)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://akamai.publicapis.solarsystems.software/v1/script",
		bytes.NewBuffer(encodedSrc),
	)
	if err != nil {
		return nil, err
	}
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

	var resp DynamicScriptResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, ApiOperationError{
			StatusCode: response.StatusCode,
			Message:    resp.Message,
		}
	}

	return &resp, nil
}
