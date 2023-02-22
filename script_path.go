package akamai

import "regexp"

var scriptPathExpr = regexp.MustCompile(`<script type="text/javascript"(?i:.*) src="((?i)[/\w\-]+)">`)

// GetScriptPath gets the Akamai Bot Manager web SDK path from the given HTML code src.
// The result is an empty string if the path was not found.
func GetScriptPath(src []byte) string {
	matches := scriptPathExpr.FindSubmatch(src)
	if len(matches) < 2 {
		return ""
	} else {
		return string(matches[1])
	}
}
