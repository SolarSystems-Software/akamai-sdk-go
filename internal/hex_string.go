package internal

import (
	"strconv"
	"strings"
)

// FromHexString converts a string with hexadecimal characters to normal characters.
func FromHexString(s string) (string, error) {
	var builder strings.Builder
	parts := strings.Split(s, "\\")[1:] // first element is always a blank string
	for _, char := range parts {
		i, err := strconv.ParseInt(strings.Replace(char, "x", "", 1), 16, 64)
		if err != nil {
			return "", err
		}
		builder.WriteRune(rune(i))
	}
	return builder.String(), nil
}
