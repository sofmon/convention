package localized

import (
	"encoding/json"
	"testing"
)

// Cross-language test fixtures - KEEP IN SYNC with localized_test.dart
const (
	jsonSingleLocale    = `{"en_GB":"Hello"}`
	jsonMultipleLocales = `{"en_GB":"Hello","nl_NL":"Hallo"}`
	jsonEmpty           = `{}`
	jsonUnicode         = `{"en_GB":"Hello","zh_CN":"你好"}`
	jsonEmptyValue      = `{"en_GB":""}`
)

func TestLocalized_Deserialize_SingleLocale(t *testing.T) {
	var l Localized
	err := json.Unmarshal([]byte(jsonSingleLocale), &l)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if l[EN] != "Hello" {
		t.Fatalf("Expected 'Hello', got '%s'", l[EN])
	}
}

func TestLocalized_Deserialize_MultipleLocales(t *testing.T) {
	var l Localized
	err := json.Unmarshal([]byte(jsonMultipleLocales), &l)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if l[EN] != "Hello" {
		t.Fatalf("Expected 'Hello' for EN, got '%s'", l[EN])
	}
	if l[NL] != "Hallo" {
		t.Fatalf("Expected 'Hallo' for NL, got '%s'", l[NL])
	}
}

func TestLocalized_Deserialize_Empty(t *testing.T) {
	var l Localized
	err := json.Unmarshal([]byte(jsonEmpty), &l)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(l) != 0 {
		t.Fatalf("Expected empty map, got %d entries", len(l))
	}
}

func TestLocalized_Deserialize_Unicode(t *testing.T) {
	var l Localized
	err := json.Unmarshal([]byte(jsonUnicode), &l)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if l[EN] != "Hello" {
		t.Fatalf("Expected 'Hello' for EN, got '%s'", l[EN])
	}
	if l[Locale("zh_CN")] != "你好" {
		t.Fatalf("Expected '你好' for zh_CN, got '%s'", l[Locale("zh_CN")])
	}
}

func TestLocalized_Deserialize_EmptyValue(t *testing.T) {
	var l Localized
	err := json.Unmarshal([]byte(jsonEmptyValue), &l)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if l[EN] != "" {
		t.Fatalf("Expected empty string for EN, got '%s'", l[EN])
	}
}

func TestLocalized_Serialize_SingleLocale(t *testing.T) {
	l := Localized{EN: "Hello"}

	data, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Compare by re-parsing (order-independent)
	var expected, actual map[string]string
	if err := json.Unmarshal([]byte(jsonSingleLocale), &expected); err != nil {
		t.Fatalf("Failed to parse expected: %v", err)
	}
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatalf("Failed to parse actual: %v", err)
	}

	if len(expected) != len(actual) {
		t.Fatalf("Length mismatch: expected %d, got %d", len(expected), len(actual))
	}
	for k, v := range expected {
		if actual[k] != v {
			t.Fatalf("Value mismatch for key %s: expected %s, got %s", k, v, actual[k])
		}
	}
}

func TestLocalized_Serialize_MultipleLocales(t *testing.T) {
	l := Localized{EN: "Hello", NL: "Hallo"}

	data, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var expected, actual map[string]string
	if err := json.Unmarshal([]byte(jsonMultipleLocales), &expected); err != nil {
		t.Fatalf("Failed to parse expected: %v", err)
	}
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatalf("Failed to parse actual: %v", err)
	}

	if len(expected) != len(actual) {
		t.Fatalf("Length mismatch")
	}
	for k, v := range expected {
		if actual[k] != v {
			t.Fatalf("Value mismatch for key %s", k)
		}
	}
}

func TestLocalized_Serialize_Empty(t *testing.T) {
	l := Localized{}

	data, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Empty map serializes to null in Go, so we check both
	if string(data) != jsonEmpty && string(data) != "null" {
		t.Fatalf("Expected %s or null, got %s", jsonEmpty, string(data))
	}
}

func TestLocalized_RoundTrip(t *testing.T) {
	original := Localized{EN: "Hello", NL: "Hallo"}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Localized
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(original) != len(restored) {
		t.Fatalf("Length mismatch: expected %d, got %d", len(original), len(restored))
	}
	for k, v := range original {
		if restored[k] != v {
			t.Fatalf("Value mismatch for key %s: expected %s, got %s", k, v, restored[k])
		}
	}
}

func TestLocalized_ValueScan_RoundTrip(t *testing.T) {
	original := Localized{EN: "Hello", NL: "Hallo"}

	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}

	var restored Localized
	err = restored.Scan(value)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	for k, v := range original {
		if restored[k] != v {
			t.Fatalf("Value mismatch for key %s: expected %s, got %s", k, v, restored[k])
		}
	}
}

func TestLocale_IsValid(t *testing.T) {
	valid := []Locale{"en_GB", "nl_NL", "de_DE", "fr_FR", "zh_CN", "ja_JP"}
	invalid := []Locale{"invalid", "xx_XX", "EN_gb", ""}

	for _, loc := range valid {
		if !loc.IsValid() {
			t.Errorf("Expected %q to be valid", loc)
		}
	}

	for _, loc := range invalid {
		if loc.IsValid() {
			t.Errorf("Expected %q to be invalid", loc)
		}
	}
}

func TestLocalized_Copy(t *testing.T) {
	original := Localized{EN: "Hello", NL: "Hallo"}
	copied := original.Copy()

	// Verify copy has same values
	for k, v := range original {
		if copied[k] != v {
			t.Fatalf("Copy mismatch for key %s", k)
		}
	}

	// Verify modifying copy doesn't affect original
	copied[EN] = "Modified"
	if original[EN] == "Modified" {
		t.Fatal("Modifying copy affected original")
	}
}
