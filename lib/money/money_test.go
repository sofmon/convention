package types

import (
	"encoding/json"
	"testing"
)

// Cross-language test fixtures - KEEP IN SYNC with money_test.dart
const (
	jsonEUR      = `{"amount":12.34,"currency":"EUR"}`
	jsonJPY      = `{"amount":567,"currency":"JPY"}`
	jsonUSD      = `{"amount":100,"currency":"USD"}`
	jsonZero     = `{"amount":0,"currency":"EUR"}`
	jsonNegative = `{"amount":-50.25,"currency":"EUR"}`

	strEUR = "12.34 EUR"
	strJPY = "567 JPY"
	strUSD = "100.00 USD"
)

// Currency helper tests

func TestMoney_EUR_FromCents(t *testing.T) {
	m := EUR.FromCents(1234)
	if m.String() != strEUR {
		t.Fatalf("Expected %q, got %q", strEUR, m.String())
	}
}

func TestMoney_EUR_FromEuro(t *testing.T) {
	m := EUR.FromEuro(100)
	if m.String() != "100.00 EUR" {
		t.Fatalf("Expected '100.00 EUR', got %q", m.String())
	}
}

func TestMoney_USD_FromCents(t *testing.T) {
	m := USD.FromCents(10000)
	if m.String() != strUSD {
		t.Fatalf("Expected %q, got %q", strUSD, m.String())
	}
}

func TestMoney_USD_FromDollars(t *testing.T) {
	m := USD.FromDollars(50)
	if m.String() != "50.00 USD" {
		t.Fatalf("Expected '50.00 USD', got %q", m.String())
	}
}

func TestMoney_JPY_FromYen(t *testing.T) {
	m := JPY.FromYen(567)
	if m.String() != strJPY {
		t.Fatalf("Expected %q, got %q", strJPY, m.String())
	}
}

// Arithmetic tests

func TestMoney_Add(t *testing.T) {
	a := EUR.FromCents(1000)
	b := EUR.FromCents(234)
	result := a.Add(b)
	if result.String() != strEUR {
		t.Fatalf("Expected %q, got %q", strEUR, result.String())
	}
}

func TestMoney_Sub(t *testing.T) {
	a := EUR.FromCents(2000)
	b := EUR.FromCents(766)
	result := a.Sub(b)
	if result.String() != strEUR {
		t.Fatalf("Expected %q, got %q", strEUR, result.String())
	}
}

func TestMoney_Mul(t *testing.T) {
	m := EUR.FromCents(617)
	result := m.Mul(2)
	if result.String() != strEUR {
		t.Fatalf("Expected %q, got %q", strEUR, result.String())
	}
}

func TestMoney_Mul_Rounding(t *testing.T) {
	m := EUR.FromCents(100) // 1.00 EUR
	result := m.Mul(0.333)  // 0.333 EUR -> 33 cents (rounded)
	if result.String() != "0.33 EUR" {
		t.Fatalf("Expected '0.33 EUR', got %q", result.String())
	}
}

func TestMoney_Div(t *testing.T) {
	m := EUR.FromCents(2468)
	result := m.Div(2)
	if result.String() != strEUR {
		t.Fatalf("Expected %q, got %q", strEUR, result.String())
	}
}

func TestMoney_Div_Rounding(t *testing.T) {
	m := EUR.FromCents(100) // 1.00 EUR
	result := m.Div(3)      // 0.333... EUR -> 33 cents (rounded)
	if result.String() != "0.33 EUR" {
		t.Fatalf("Expected '0.33 EUR', got %q", result.String())
	}
}

func TestMoney_Add_CurrencyMismatch_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic on currency mismatch")
		}
	}()

	a := EUR.FromCents(100)
	b := USD.FromCents(100)
	a.Add(b) // Should panic
}

func TestMoney_Sub_CurrencyMismatch_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic on currency mismatch")
		}
	}()

	a := EUR.FromCents(100)
	b := JPY.FromYen(100)
	a.Sub(b) // Should panic
}

func TestMoney_Div_ByZero_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic on division by zero")
		}
	}()

	m := EUR.FromCents(100)
	m.Div(0) // Should panic
}

// String tests

func TestMoney_String_EUR(t *testing.T) {
	m := EUR.FromCents(1234)
	if m.String() != strEUR {
		t.Fatalf("Expected %q, got %q", strEUR, m.String())
	}
}

func TestMoney_String_JPY(t *testing.T) {
	m := JPY.FromYen(567)
	if m.String() != strJPY {
		t.Fatalf("Expected %q, got %q", strJPY, m.String())
	}
}

func TestMoney_String_Zero(t *testing.T) {
	m := EUR.FromCents(0)
	if m.String() != "0.00 EUR" {
		t.Fatalf("Expected '0.00 EUR', got %q", m.String())
	}
}

func TestMoney_String_Negative(t *testing.T) {
	m := EUR.FromCents(-5025)
	if m.String() != "-50.25 EUR" {
		t.Fatalf("Expected '-50.25 EUR', got %q", m.String())
	}
}

// Parse tests

func TestMoney_Parse_EUR(t *testing.T) {
	m, err := Parse(strEUR)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if m.String() != strEUR {
		t.Fatalf("Expected %q, got %q", strEUR, m.String())
	}
}

func TestMoney_Parse_JPY(t *testing.T) {
	m, err := Parse(strJPY)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if m.String() != strJPY {
		t.Fatalf("Expected %q, got %q", strJPY, m.String())
	}
}

func TestMoney_Parse_USD(t *testing.T) {
	m, err := Parse(strUSD)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if m.String() != strUSD {
		t.Fatalf("Expected %q, got %q", strUSD, m.String())
	}
}

func TestMoney_Parse_InvalidFormat(t *testing.T) {
	_, err := Parse("invalid")
	if err == nil {
		t.Fatal("Expected error for invalid format")
	}
}

func TestMoney_Parse_InvalidAmount(t *testing.T) {
	_, err := Parse("abc EUR")
	if err == nil {
		t.Fatal("Expected error for invalid amount")
	}
}

func TestMoney_Parse_UnknownCurrency(t *testing.T) {
	_, err := Parse("100.00 XXX")
	if err == nil {
		t.Fatal("Expected error for unknown currency")
	}
}

// JSON Marshal tests

func TestMoney_MarshalJSON_EUR(t *testing.T) {
	m := EUR.FromCents(1234)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Compare by re-parsing (order-independent)
	var expected, actual map[string]interface{}
	if err := json.Unmarshal([]byte(jsonEUR), &expected); err != nil {
		t.Fatalf("Failed to parse expected: %v", err)
	}
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatalf("Failed to parse actual: %v", err)
	}

	if expected["currency"] != actual["currency"] {
		t.Fatalf("Currency mismatch: expected %v, got %v", expected["currency"], actual["currency"])
	}
	if expected["amount"] != actual["amount"] {
		t.Fatalf("Amount mismatch: expected %v, got %v", expected["amount"], actual["amount"])
	}
}

func TestMoney_MarshalJSON_JPY(t *testing.T) {
	m := JPY.FromYen(567)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var expected, actual map[string]interface{}
	if err := json.Unmarshal([]byte(jsonJPY), &expected); err != nil {
		t.Fatalf("Failed to parse expected: %v", err)
	}
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatalf("Failed to parse actual: %v", err)
	}

	if expected["currency"] != actual["currency"] {
		t.Fatalf("Currency mismatch: expected %v, got %v", expected["currency"], actual["currency"])
	}
	if expected["amount"] != actual["amount"] {
		t.Fatalf("Amount mismatch: expected %v, got %v", expected["amount"], actual["amount"])
	}
}

func TestMoney_MarshalJSON_Zero(t *testing.T) {
	m := EUR.FromCents(0)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var actual map[string]interface{}
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatalf("Failed to parse actual: %v", err)
	}

	if actual["amount"].(float64) != 0 {
		t.Fatalf("Expected amount 0, got %v", actual["amount"])
	}
}

func TestMoney_MarshalJSON_Negative(t *testing.T) {
	m := EUR.FromCents(-5025)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var expected, actual map[string]interface{}
	if err := json.Unmarshal([]byte(jsonNegative), &expected); err != nil {
		t.Fatalf("Failed to parse expected: %v", err)
	}
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatalf("Failed to parse actual: %v", err)
	}

	if expected["amount"] != actual["amount"] {
		t.Fatalf("Amount mismatch: expected %v, got %v", expected["amount"], actual["amount"])
	}
}

// JSON Unmarshal tests

func TestMoney_UnmarshalJSON_EUR(t *testing.T) {
	var m Money
	err := json.Unmarshal([]byte(jsonEUR), &m)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if m.String() != strEUR {
		t.Fatalf("Expected %q, got %q", strEUR, m.String())
	}
}

func TestMoney_UnmarshalJSON_JPY(t *testing.T) {
	var m Money
	err := json.Unmarshal([]byte(jsonJPY), &m)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if m.String() != strJPY {
		t.Fatalf("Expected %q, got %q", strJPY, m.String())
	}
}

func TestMoney_UnmarshalJSON_USD(t *testing.T) {
	var m Money
	err := json.Unmarshal([]byte(jsonUSD), &m)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if m.String() != strUSD {
		t.Fatalf("Expected %q, got %q", strUSD, m.String())
	}
}

func TestMoney_UnmarshalJSON_Zero(t *testing.T) {
	var m Money
	err := json.Unmarshal([]byte(jsonZero), &m)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if m.String() != "0.00 EUR" {
		t.Fatalf("Expected '0.00 EUR', got %q", m.String())
	}
}

func TestMoney_UnmarshalJSON_Negative(t *testing.T) {
	var m Money
	err := json.Unmarshal([]byte(jsonNegative), &m)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if m.String() != "-50.25 EUR" {
		t.Fatalf("Expected '-50.25 EUR', got %q", m.String())
	}
}

func TestMoney_UnmarshalJSON_UnknownCurrency(t *testing.T) {
	var m Money
	err := json.Unmarshal([]byte(`{"amount":100,"currency":"XXX"}`), &m)
	if err == nil {
		t.Fatal("Expected error for unknown currency")
	}
}

func TestMoney_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var m Money
	err := json.Unmarshal([]byte(`invalid`), &m)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

// Round-trip tests

func TestMoney_RoundTrip_String(t *testing.T) {
	original := EUR.FromCents(1234)

	str := original.String()
	restored, err := Parse(str)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if original.String() != restored.String() {
		t.Fatalf("Round-trip failed: expected %q, got %q", original.String(), restored.String())
	}
}

func TestMoney_RoundTrip_JSON(t *testing.T) {
	original := EUR.FromCents(1234)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Money
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if original.String() != restored.String() {
		t.Fatalf("Round-trip failed: expected %q, got %q", original.String(), restored.String())
	}
}

func TestMoney_RoundTrip_JSON_JPY(t *testing.T) {
	original := JPY.FromYen(567)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Money
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if original.String() != restored.String() {
		t.Fatalf("Round-trip failed: expected %q, got %q", original.String(), restored.String())
	}
}
