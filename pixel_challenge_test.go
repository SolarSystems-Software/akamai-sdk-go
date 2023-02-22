package akamai

import "testing"

func TestGetPixelChallengeHtmlVar(t *testing.T) {
	const (
		validInput   = `bazadebezolkohpepadr="500"`
		invalidInput = `bazadebezolkohpepadr="a"`
	)

	if v, err := GetPixelChallengeHtmlVar([]byte(validInput)); err != nil {
		t.Fatal("err != nil on valid input:", err)
	} else if v != 500 {
		t.Fatal("v != 500:", v)
	}

	if _, err := GetPixelChallengeHtmlVar([]byte(invalidInput)); err == nil {
		t.Fatal("err == nil on valid input")
	}
}
