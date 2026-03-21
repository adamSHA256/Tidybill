package email

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	gomail "github.com/wneessen/go-mail"

	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/model"
)

// Encryption salt for SMTP password storage
const encryptionSalt = "tidybill-smtp-v1"

type Service struct {
	smtpRepo     *repository.SmtpConfigRepository
	invoiceRepo  *repository.InvoiceRepository
	customerRepo *repository.CustomerRepository
	supplierRepo *repository.SupplierRepository
	settingsRepo *repository.SettingsRepository
}

func NewService(
	smtpRepo *repository.SmtpConfigRepository,
	invoiceRepo *repository.InvoiceRepository,
	customerRepo *repository.CustomerRepository,
	supplierRepo *repository.SupplierRepository,
	settingsRepo *repository.SettingsRepository,
) *Service {
	return &Service{
		smtpRepo:     smtpRepo,
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
		supplierRepo: supplierRepo,
		settingsRepo: settingsRepo,
	}
}

// loadInvoiceWithRelations loads an invoice and populates its Customer and Supplier relations.
func (s *Service) loadInvoiceWithRelations(invoiceID string) (*model.Invoice, error) {
	inv, err := s.invoiceRepo.GetByID(invoiceID)
	if err != nil {
		return nil, fmt.Errorf("load invoice: %w", err)
	}
	if inv == nil {
		return nil, fmt.Errorf("invoice not found")
	}

	cust, err := s.customerRepo.GetByID(inv.CustomerID)
	if err == nil && cust != nil {
		inv.Customer = cust
	}

	sup, err := s.supplierRepo.GetByID(inv.SupplierID)
	if err == nil && sup != nil {
		inv.Supplier = sup
	}

	return inv, nil
}

// SendInvoiceEmail sends an invoice PDF via email and updates the invoice.
func (s *Service) SendInvoiceEmail(invoiceID, to, subject, body string, sendCopy bool) error {
	inv, err := s.loadInvoiceWithRelations(invoiceID)
	if err != nil {
		return err
	}

	// Check PDF exists
	if inv.PDFPath == "" {
		return fmt.Errorf("invoice has no PDF generated")
	}
	if _, err := os.Stat(inv.PDFPath); os.IsNotExist(err) {
		return fmt.Errorf("PDF file not found: %s", inv.PDFPath)
	}

	// Load SMTP config
	config, err := s.smtpRepo.GetBySupplierID(inv.SupplierID)
	if err != nil {
		return fmt.Errorf("load SMTP config: %w", err)
	}
	if config == nil {
		return fmt.Errorf("SMTP not configured for this supplier")
	}

	// Decrypt password
	password, err := DecryptPassword(config.PasswordEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt SMTP password: %w", err)
	}

	// Send email
	if err := s.send(config, password, to, subject, body, inv.PDFPath, inv.InvoiceNumber+".pdf"); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	// Send a copy to the sender's own inbox for their records
	if sendCopy {
		copySubjectTemplate, _ := s.settingsRepo.Get("email.copy_subject")
		if copySubjectTemplate == "" {
			copySubjectTemplate = "TidyBill - ((subject))"
		}
		copySubject := strings.ReplaceAll(copySubjectTemplate, "((subject))", subject)
		_ = s.send(config, password, config.FromEmail, copySubject, body, inv.PDFPath, inv.InvoiceNumber+".pdf")
		// Ignore errors on copy — the main send succeeded
	}

	// Update invoice: set email_sent_at and status to "sent" if draft/created
	if err := s.invoiceRepo.SetEmailSentAt(invoiceID); err != nil {
		return fmt.Errorf("update email_sent_at: %w", err)
	}

	if inv.Status == model.StatusDraft || inv.Status == model.StatusCreated {
		if err := s.invoiceRepo.UpdateStatus(invoiceID, model.StatusSent); err != nil {
			// Non-fatal: email was sent but status update failed
			fmt.Printf("warning: email sent but status update failed: %v\n", err)
		}
	}

	return nil
}

// TestConnection tests SMTP connection by sending a test email.
func (s *Service) TestConnection(config *model.SmtpConfig, password string) error {
	m := gomail.NewMsg()
	if err := m.From(config.FromEmail); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if err := m.To(config.FromEmail); err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	m.Subject("TidyBill - Test Connection")
	m.SetBodyString(gomail.TypeTextPlain, "This is a test email from TidyBill. Your SMTP configuration is working correctly.")

	return s.dial(config, password, m)
}

// GetEmailPreview returns the pre-filled email data for an invoice.
func (s *Service) GetEmailPreview(invoiceID string) (*EmailPreview, error) {
	inv, err := s.loadInvoiceWithRelations(invoiceID)
	if err != nil {
		return nil, err
	}

	// Check if SMTP is configured
	smtpConfig, err := s.smtpRepo.GetBySupplierID(inv.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("load SMTP config: %w", err)
	}

	// Get effective template (customer-specific or global)
	subject, body, err := s.getEffectiveTemplate(inv)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	// Replace placeholders
	subject = s.ReplacePlaceholders(subject, inv)
	body = s.ReplacePlaceholders(body, inv)

	// Get customer email
	customerEmail := ""
	if inv.Customer != nil {
		customerEmail = inv.Customer.Email
	}

	preview := &EmailPreview{
		To:            customerEmail,
		Subject:       subject,
		Body:          body,
		PDFFilename:   inv.InvoiceNumber + ".pdf",
		HasSmtp:       smtpConfig != nil && smtpConfig.HasPassword,
		AlreadySentAt: inv.EmailSentAt,
	}

	return preview, nil
}

// EmailPreview holds the pre-filled email data for an invoice.
type EmailPreview struct {
	To            string     `json:"to"`
	Subject       string     `json:"subject"`
	Body          string     `json:"body"`
	PDFFilename   string     `json:"pdf_filename"`
	HasSmtp       bool       `json:"has_smtp"`
	AlreadySentAt *time.Time `json:"already_sent_at"`
}

// ReplacePlaceholders replaces ((placeholder)) tokens in a template string.
func (s *Service) ReplacePlaceholders(template string, inv *model.Invoice) string {
	r := strings.NewReplacer(
		"((number))", inv.InvoiceNumber,
		"((total))", formatTotal(inv.Total, inv.Currency),
		"((due_date))", inv.DueDate.Format("02.01.2006"),
		"((issue_date))", inv.IssueDate.Format("02.01.2006"),
	)
	result := r.Replace(template)

	if inv.Customer != nil {
		result = strings.ReplaceAll(result, "((customer))", inv.Customer.Name)
	}
	if inv.Supplier != nil {
		result = strings.ReplaceAll(result, "((supplier))", inv.Supplier.Name)
	}

	return result
}

func formatTotal(total float64, currency string) string {
	// Format with 2 decimal places and space before currency
	return fmt.Sprintf("%.2f %s", total, currency)
}

// getEffectiveTemplate returns the subject/body template to use for an invoice.
func (s *Service) getEffectiveTemplate(inv *model.Invoice) (subject, body string, err error) {
	// Check if customer has custom template
	if inv.Customer != nil && inv.Customer.EmailCustomTemplate {
		return inv.Customer.EmailSubjectTemplate, inv.Customer.EmailBodyTemplate, nil
	}

	// Fall back to global settings
	subject, _ = s.settingsRepo.Get("email.default_subject")
	body, _ = s.settingsRepo.Get("email.default_body")

	// Fall back to hardcoded defaults
	if subject == "" {
		subject = "Faktura ((number))"
	}
	if body == "" {
		body = "Dobr\u00fd den,\n\nv p\u0159\u00edloze zas\u00edl\u00e1m fakturu \u010d. ((number)) na \u010d\u00e1stku ((total)).\nSplatnost: ((due_date)).\n\nS pozdravem\n((supplier))"
	}

	return subject, body, nil
}

func (s *Service) send(config *model.SmtpConfig, password, to, subject, body, pdfPath, pdfFilename string) error {
	m := gomail.NewMsg()
	if err := m.FromFormat(config.FromName, config.FromEmail); err != nil {
		return fmt.Errorf("invalid from: %w", err)
	}
	if err := m.To(to); err != nil {
		return fmt.Errorf("invalid to: %w", err)
	}
	m.Subject(subject)
	m.SetBodyString(gomail.TypeTextPlain, body)
	m.AttachFile(pdfPath, gomail.WithFileName(pdfFilename))

	return s.dial(config, password, m)
}

func (s *Service) dial(config *model.SmtpConfig, password string, m *gomail.Msg) error {
	opts := []gomail.Option{
		gomail.WithPort(config.Port),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(config.Username),
		gomail.WithPassword(password),
		gomail.WithTimeout(30 * time.Second),
	}

	if config.UseStartTLS {
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.TLSMandatory))
	} else {
		opts = append(opts, gomail.WithSSL())
	}

	c, err := gomail.NewClient(config.Host, opts...)
	if err != nil {
		return fmt.Errorf("SMTP client: %w", err)
	}
	return c.DialAndSend(m)
}

// EncryptPassword encrypts a password for storage.
func EncryptPassword(plaintext string) (string, error) {
	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptPassword decrypts a stored password.
func DecryptPassword(encrypted string) (string, error) {
	if encrypted == "" {
		return "", fmt.Errorf("no password stored")
	}
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func deriveKey() []byte {
	// Use hostname as machine-specific component
	hostname, _ := os.Hostname()
	h := sha256.Sum256([]byte(encryptionSalt + hostname))
	return h[:]
}
