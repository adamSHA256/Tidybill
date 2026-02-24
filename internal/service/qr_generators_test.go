package service

import (
	"strings"
	"testing"
	"time"

	"github.com/adamSHA256/tidybill/internal/model"
)

func makeTestData() *InvoiceData {
	return &InvoiceData{
		Invoice: &model.Invoice{
			InvoiceNumber:  "FV2024001",
			Total:          1250.00,
			Currency:       "EUR",
			VariableSymbol: "2024001",
			DueDate:        time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		Supplier: &model.Supplier{
			Name:   "Test Company s.r.o.",
			Street: "Hlavná 1",
			City:   "Bratislava",
		},
		Customer: &model.Customer{},
		BankAccount: &model.BankAccount{
			IBAN:   "SK8209000000002918599669",
			SWIFT:  "GIBASKBX",
			QRType: "spayd",
		},
	}
}

// ── SPAYD ────────────────────────────────────────────────────────────────

func TestSPAYD_Basic(t *testing.T) {
	data := makeTestData()
	result := generateSPAYD(data)

	if !strings.HasPrefix(result, "SPD*1.0*") {
		t.Errorf("expected SPD*1.0* prefix, got: %s", result)
	}
	if !strings.Contains(result, "ACC:SK8209000000002918599669") {
		t.Error("expected IBAN in result")
	}
	if !strings.Contains(result, "AM:1250.00") {
		t.Error("expected amount 1250.00")
	}
	if !strings.Contains(result, "CC:EUR") {
		t.Error("expected currency EUR")
	}
	if !strings.Contains(result, "X-VS:2024001") {
		t.Error("expected variable symbol")
	}
}

func TestSPAYD_NoIBAN(t *testing.T) {
	data := makeTestData()
	data.BankAccount.IBAN = ""
	if result := generateSPAYD(data); result != "" {
		t.Errorf("expected empty string for missing IBAN, got: %s", result)
	}
}

func TestSPAYD_CZK(t *testing.T) {
	data := makeTestData()
	data.Invoice.Currency = "CZK"
	result := generateSPAYD(data)
	if !strings.Contains(result, "CC:CZK") {
		t.Errorf("expected CC:CZK, got: %s", result)
	}
}

func TestSPAYD_StripSpacesFromIBAN(t *testing.T) {
	data := makeTestData()
	data.BankAccount.IBAN = "SK82 0900 0000 0029 1859 9669"
	result := generateSPAYD(data)
	if !strings.Contains(result, "ACC:SK8209000000002918599669") {
		t.Errorf("expected spaces stripped from IBAN, got: %s", result)
	}
}

// ── EPC ──────────────────────────────────────────────────────────────────

func TestEPC_Basic(t *testing.T) {
	data := makeTestData()
	result := generateEPC(data)

	lines := strings.Split(result, "\n")
	if len(lines) < 8 {
		t.Fatalf("expected at least 8 lines, got %d: %s", len(lines), result)
	}
	if lines[0] != "BCD" {
		t.Errorf("line 1: expected BCD, got %s", lines[0])
	}
	if lines[1] != "002" {
		t.Errorf("line 2: expected 002, got %s", lines[1])
	}
	if lines[2] != "1" {
		t.Errorf("line 3: expected 1 (UTF-8), got %s", lines[2])
	}
	if lines[3] != "SCT" {
		t.Errorf("line 4: expected SCT, got %s", lines[3])
	}
	if lines[4] != "GIBASKBX" {
		t.Errorf("line 5: expected BIC, got %s", lines[4])
	}
	if lines[5] != "Test Company s.r.o." {
		t.Errorf("line 6: expected beneficiary name, got %s", lines[5])
	}
	if lines[6] != "SK8209000000002918599669" {
		t.Errorf("line 7: expected IBAN, got %s", lines[6])
	}
	if lines[7] != "EUR1250.00" {
		t.Errorf("line 8: expected EUR1250.00, got %s", lines[7])
	}
}

func TestEPC_NonEUR_ReturnsEmpty(t *testing.T) {
	data := makeTestData()
	data.Invoice.Currency = "CZK"
	if result := generateEPC(data); result != "" {
		t.Errorf("EPC should return empty for non-EUR currency, got: %s", result)
	}
}

func TestEPC_NoIBAN_ReturnsEmpty(t *testing.T) {
	data := makeTestData()
	data.BankAccount.IBAN = ""
	if result := generateEPC(data); result != "" {
		t.Errorf("EPC should return empty for missing IBAN, got: %s", result)
	}
}

func TestEPC_NoBIC(t *testing.T) {
	data := makeTestData()
	data.BankAccount.SWIFT = ""
	result := generateEPC(data)
	lines := strings.Split(result, "\n")
	if len(lines) < 5 {
		t.Fatal("expected at least 5 lines")
	}
	if lines[4] != "" {
		t.Errorf("line 5 (BIC) should be empty for v002, got: %s", lines[4])
	}
}

func TestEPC_Remittance(t *testing.T) {
	data := makeTestData()
	result := generateEPC(data)
	// Should contain VS+number and invoice number in remittance (last non-empty line)
	if !strings.Contains(result, "VS2024001") {
		t.Error("expected variable symbol in remittance")
	}
	if !strings.Contains(result, "FV2024001") {
		t.Error("expected invoice number in remittance")
	}
}

func TestEPC_CaseInsensitiveCurrency(t *testing.T) {
	data := makeTestData()
	data.Invoice.Currency = "eur"
	result := generateEPC(data)
	if result == "" {
		t.Error("EPC should accept lowercase 'eur'")
	}
}

// ── Pay by Square ────────────────────────────────────────────────────────

func TestPayBySquare_Basic(t *testing.T) {
	data := makeTestData()
	result := generatePayBySquare(data)

	if result == "" {
		t.Fatal("expected non-empty Pay by Square string")
	}
	// Pay by Square output is Base32hex encoded — should only contain 0-9A-V
	for _, c := range result {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'V')) {
			t.Errorf("unexpected character in Base32hex output: %c", c)
			break
		}
	}
}

func TestPayBySquare_NoIBAN_ReturnsEmpty(t *testing.T) {
	data := makeTestData()
	data.BankAccount.IBAN = ""
	if result := generatePayBySquare(data); result != "" {
		t.Errorf("Pay by Square should return empty for missing IBAN, got: %s", result)
	}
}

func TestPayBySquare_DifferentCurrencies(t *testing.T) {
	// Pay by Square supports any currency (unlike EPC which is EUR-only)
	for _, curr := range []string{"EUR", "CZK", "USD"} {
		data := makeTestData()
		data.Invoice.Currency = curr
		result := generatePayBySquare(data)
		if result == "" {
			t.Errorf("Pay by Square should work with %s currency", curr)
		}
	}
}

func TestPayBySquare_Deterministic(t *testing.T) {
	data := makeTestData()
	r1 := generatePayBySquare(data)
	r2 := generatePayBySquare(data)
	if r1 != r2 {
		t.Error("Pay by Square should produce deterministic output for same input")
	}
}

// ── GenerateQRPayload dispatch ───────────────────────────────────────────

func TestGenerateQRPayload_Dispatch(t *testing.T) {
	data := makeTestData()

	if r := GenerateQRPayload("spayd", data); !strings.HasPrefix(r, "SPD*1.0*") {
		t.Error("spayd dispatch failed")
	}
	if r := GenerateQRPayload("epc", data); !strings.HasPrefix(r, "BCD\n") {
		t.Error("epc dispatch failed")
	}
	if r := GenerateQRPayload("pay_by_square", data); r == "" {
		t.Error("pay_by_square dispatch failed")
	}
	if r := GenerateQRPayload("none", data); r != "" {
		t.Error("none should return empty")
	}
	if r := GenerateQRPayload("unknown", data); r != "" {
		t.Error("unknown type should return empty")
	}
}
