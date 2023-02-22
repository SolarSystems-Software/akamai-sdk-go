package akamai

import (
	"os"
	"testing"
)

// TestIsScriptStatic tests the output of IsScriptStatic with a dynamic script and
// static script as input.
func TestIsScriptStatic(t *testing.T) {
	dynamicInput, err := os.ReadFile("tests/dynamic_script_input.js")
	if err != nil {
		t.Error(err)
	}

	staticInput, err := os.ReadFile("tests/static_script_input.js")
	if err != nil {
		t.Error(err)
	}

	if IsScriptStatic(dynamicInput) {
		t.Fail()
	}

	if !IsScriptStatic(staticInput) {
		t.Fail()
	}
}

// BenchmarkIsScriptStaticWithDynamicScript benchmarks IsScriptStatic with a dynamic script as input.
func BenchmarkIsScriptStaticWithDynamicScript(b *testing.B) {
	input, err := os.ReadFile("tests/dynamic_script_input.js")
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		IsScriptStatic(input)
	}
}

// BenchmarkIsScriptStaticWithStaticScript benchmarks IsScriptStatic with a static script as input.
func BenchmarkIsScriptStaticWithStaticScript(b *testing.B) {
	input, err := os.ReadFile("tests/static_script_input.js")
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		IsScriptStatic(input)
	}
}
