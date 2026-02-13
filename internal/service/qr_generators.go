package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/adamSHA256/tidybill/internal/i18n"
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

// generateSPAYD creates Czech SPAYD QR payment string
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

// generatePayBySquare creates Slovak Pay by Square QR string
// TODO: Implement full Pay by Square encoding (requires LZMA compression)
func generatePayBySquare(data *InvoiceData) string {
	log.Println("Pay by Square not yet implemented, falling back to SPAYD")
	return generateSPAYD(data)
}

// generateEPC creates EU EPC QR / GiroCode string
// TODO: Implement full EPC encoding
func generateEPC(data *InvoiceData) string {
	log.Println("EPC QR not yet implemented, falling back to SPAYD")
	return generateSPAYD(data)
}
