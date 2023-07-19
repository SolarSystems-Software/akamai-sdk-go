package akamai

import "regexp"

var scriptPathExpr = regexp.MustCompile(`<script type="text/javascript"(?i:.*) src="((?i)[/\w\-]+)">`)

// GetScriptPath gets the Akamai Bot Manager web SDK path from the given HTML code src.
// ok is true if the path was found, otherwise it is false.
func GetScriptPath(src []byte) (ok bool, path string) {
	matches := scriptPathExpr.FindSubmatch(src)
	if ok = len(matches) >= 2; ok {
		path = string(matches[1])
	}
	return
}
