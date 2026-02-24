package service

import (
	"bytes"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"strings"

	"github.com/adamSHA256/tidybill/internal/i18n"
	"github.com/ulikunitz/xz/lzma"
)

// GenerateQRPayload generates QR payment string based on type
func GenerateQRPayload(qrType string, data *InvoiceData) string {
	switch qrType {
	case "spayd":
		return generateSPAYD(data)
	case "pay_by_square":
		return generatePayBySquare(data)
	case "epc":
		return generateEPC(data)
	default:
		return ""
	}
}

// generateSPAYD creates Czech SPAYD QR payment string.
// Supports any currency via the CC (Currency Code) field.
func generateSPAYD(data *InvoiceData) string {
	if data.BankAccount.IBAN == "" {
		return ""
	}
	iban := strings.ReplaceAll(data.BankAccount.IBAN, " ", "")
	return fmt.Sprintf("SPD*1.0*ACC:%s*AM:%.2f*CC:%s*X-VS:%s*MSG:%s",
		iban,
		data.Invoice.Total,
		data.Invoice.Currency,
		data.Invoice.VariableSymbol,
		i18n.Tf("pdf.invoice_msg", data.Invoice.InvoiceNumber),
	)
}

// generateEPC creates an EU EPC QR / GiroCode string (EPC069-12 v3.1).
//
// The EPC standard is EUR-only. If the invoice currency is not EUR,
// this returns an empty string and no QR code will be rendered.
//
// Format: 12 newline-separated lines:
//
//	BCD           (service tag)
//	002           (version — BIC optional within EEA)
//	1             (UTF-8 encoding)
//	SCT           (SEPA Credit Transfer)
//	<BIC>         (optional for v002)
//	<name>        (beneficiary, max 70)
//	<IBAN>        (max 34, no spaces)
//	EUR<amount>   (e.g. EUR1234.56)
//	<purpose>     (SEPA purpose code, optional)
//	<struct ref>  (creditor reference, mutually exclusive with line 11)
//	<unstruct>    (remittance info — we put variable symbol here)
//	<info>        (beneficiary-to-originator info, optional)
func generateEPC(data *InvoiceData) string {
	if data.BankAccount.IBAN == "" {
		return ""
	}
	// EPC QR codes only support EUR
	if !strings.EqualFold(data.Invoice.Currency, "EUR") {
		return ""
	}

	iban := strings.ReplaceAll(data.BankAccount.IBAN, " ", "")
	bic := strings.ReplaceAll(data.BankAccount.SWIFT, " ", "")

	// Beneficiary name — max 70 chars
	name := data.Supplier.Name
	if len(name) > 70 {
		name = name[:70]
	}

	// Unstructured remittance — use variable symbol / invoice reference, max 140
	remittance := ""
	if data.Invoice.VariableSymbol != "" {
		remittance = "VS" + data.Invoice.VariableSymbol
	}
	if data.Invoice.InvoiceNumber != "" {
		if remittance != "" {
			remittance += " "
		}
		remittance += data.Invoice.InvoiceNumber
	}
	if len(remittance) > 140 {
		remittance = remittance[:140]
	}

	// Build the 12-line payload; trailing empty lines can be omitted
	lines := []string{
		"BCD",                                   // 1: service tag
		"002",                                   // 2: version (BIC optional)
		"1",                                     // 3: UTF-8
		"SCT",                                   // 4: SEPA Credit Transfer
		bic,                                     // 5: BIC (optional for v002)
		name,                                    // 6: beneficiary name
		iban,                                    // 7: IBAN
		fmt.Sprintf("EUR%.2f", data.Invoice.Total), // 8: amount
		"",                                      // 9: purpose (empty)
		"",                                      // 10: structured reference (empty)
		remittance,                              // 11: unstructured remittance
	}
	// Trim trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

// generatePayBySquare creates a Slovak Pay by Square QR string.
//
// Encoding pipeline (per SBA specification v1.1.0):
//  1. Serialize payment data as tab-separated UTF-8 string
//  2. Prepend CRC32 checksum (IEEE, little-endian, 4 bytes)
//  3. Compress with LZMA1 (lc=3, lp=0, pb=2, dictCap=128KB), strip 13-byte header
//  4. Prepend 2-byte bysquare header + 2-byte payload length (LE)
//  5. Encode everything as Base32hex (RFC 4648, no padding)
func generatePayBySquare(data *InvoiceData) string {
	if data.BankAccount.IBAN == "" {
		return ""
	}

	iban := strings.ReplaceAll(data.BankAccount.IBAN, " ", "")
	bic := strings.ReplaceAll(data.BankAccount.SWIFT, " ", "")

	// Due date in YYYYMMDD format (optional)
	dueDate := ""
	if !data.Invoice.DueDate.IsZero() {
		dueDate = data.Invoice.DueDate.Format("20060102")
	}

	// Payment note
	note := i18n.Tf("pdf.invoice_msg", data.Invoice.InvoiceNumber)
	if len(note) > 140 {
		note = note[:140]
	}

	// Beneficiary name (max 70)
	beneficiary := data.Supplier.Name
	if len(beneficiary) > 70 {
		beneficiary = beneficiary[:70]
	}

	// Amount as string with dot decimal separator, empty if zero
	amount := ""
	if data.Invoice.Total > 0 {
		amount = fmt.Sprintf("%.2f", data.Invoice.Total)
	}

	// ── Step 1: Build tab-separated data ──
	// Field order per bysquare spec v1.1.0:
	//   invoiceId \t paymentsCount \t
	//   [payment: type, amount, currency, dueDate, variableSymbol,
	//    constantSymbol, specificSymbol, originatorsRef, paymentNote,
	//    bankAccountsCount, IBAN, BIC, standingOrderExt, directDebitExt]
	//   [beneficiary: name, street, city]
	fields := []string{
		"",                          // invoiceId (optional)
		"1",                         // paymentsCount
		"1",                         // type: PaymentOrder
		amount,                      // amount
		data.Invoice.Currency,       // currencyCode
		dueDate,                     // paymentDueDate
		data.Invoice.VariableSymbol, // variableSymbol
		"",                          // constantSymbol
		"",                          // specificSymbol
		"",                          // originatorsReferenceInformation
		note,                        // paymentNote
		"1",                         // bankAccountsCount
		iban,                        // IBAN
		bic,                         // BIC
		"0",                         // standingOrderExt
		"0",                         // directDebitExt
		beneficiary,                 // beneficiary name
		"",                          // beneficiary street
		"",                          // beneficiary city
	}

	tabData := strings.Join(fields, "\t")

	// ── Step 2: CRC32 checksum ──
	payload := []byte(tabData)
	checksum := crc32.ChecksumIEEE(payload)
	crcBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(crcBytes, checksum)
	withCRC := append(crcBytes, payload...)

	// ── Step 3: LZMA1 compression ──
	compressed, err := compressLZMA(withCRC)
	if err != nil {
		return ""
	}

	// ── Step 4: Build header + payload length ──
	// Header: 2 bytes — [bySquareType(4bit)|version(4bit)] [docType(4bit)|reserved(4bit)]
	// Version 0x01 = spec v1.1.0
	header := []byte{0x00, 0x00}
	header[0] = 0x00<<4 | 0x01 // bySquareType=0, version=1

	// Payload length: 2 bytes, little-endian, length of uncompressed CRC+payload
	lengthBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(lengthBytes, uint16(len(withCRC)))

	// ── Step 5: Concatenate and Base32hex encode ──
	var result []byte
	result = append(result, header...)
	result = append(result, lengthBytes...)
	result = append(result, compressed...)

	return base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(result)
}

// compressLZMA compresses data using LZMA1 with Pay by Square parameters
// and returns only the compressed body (13-byte LZMA header stripped).
func compressLZMA(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := lzma.WriterConfig{
		Properties: &lzma.Properties{LC: 3, LP: 0, PB: 2},
		DictCap:    1 << 17, // 128 KB
		Size:       int64(len(data)),
	}.NewWriter(&buf)
	if err != nil {
		return nil, fmt.Errorf("lzma writer: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("lzma write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("lzma close: %w", err)
	}

	// LZMA output: 13-byte header (1 props + 4 dictSize + 8 uncompressedSize) + body
	raw := buf.Bytes()
	if len(raw) <= 13 {
		return nil, fmt.Errorf("lzma output too short: %d bytes", len(raw))
	}
	return raw[13:], nil
}
