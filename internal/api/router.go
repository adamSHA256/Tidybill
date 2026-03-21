package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/email"
	"github.com/adamSHA256/tidybill/internal/service"
)

type Server struct {
	invoices     *repository.InvoiceRepository
	invoiceItems *repository.InvoiceItemRepository
	customers    *repository.CustomerRepository
	suppliers    *repository.SupplierRepository
	bankAccounts *repository.BankAccountRepository
	settings     *repository.SettingsRepository
	items        *repository.ItemRepository
	custItems    *repository.CustomerItemRepository
	templates    *repository.PDFTemplateRepository
	smtpConfigs  *repository.SmtpConfigRepository
	pdf          *service.PDFService
	emailService *email.Service
	cfg          *config.Config
}

func NewServer(db *sql.DB, cfg *config.Config) *Server {
	smtpConfigs := repository.NewSmtpConfigRepository(db)
	invoices := repository.NewInvoiceRepository(db)
	customers := repository.NewCustomerRepository(db)
	suppliers := repository.NewSupplierRepository(db)
	settings := repository.NewSettingsRepository(db)

	s := &Server{
		invoices:     invoices,
		invoiceItems: repository.NewInvoiceItemRepository(db),
		customers:    customers,
		suppliers:    suppliers,
		bankAccounts: repository.NewBankAccountRepository(db),
		settings:     settings,
		items:        repository.NewItemRepository(db),
		custItems:    repository.NewCustomerItemRepository(db),
		templates:    repository.NewPDFTemplateRepository(db),
		smtpConfigs:  smtpConfigs,
		pdf:          service.NewPDFService(cfg.PDFDir, cfg.PreviewDir),
		emailService: email.NewService(smtpConfigs, invoices, customers, suppliers, settings),
		cfg:          cfg,
	}
	return s
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Dashboard
	mux.HandleFunc("GET /api/dashboard/stats", s.getDashboardStats)

	// Invoices
	mux.HandleFunc("GET /api/invoices", s.listInvoices)
	mux.HandleFunc("POST /api/invoices", s.createInvoice)
	mux.HandleFunc("GET /api/invoices/next-number", s.getNextInvoiceNumber)
	mux.HandleFunc("GET /api/invoices/{id}", s.getInvoice)
	mux.HandleFunc("PUT /api/invoices/{id}", s.updateInvoice)
	mux.HandleFunc("DELETE /api/invoices/{id}", s.deleteInvoice)
	mux.HandleFunc("PUT /api/invoices/{id}/status", s.updateInvoiceStatus)
	mux.HandleFunc("PUT /api/invoices/{id}/notes", s.updateInvoiceNotes)
	mux.HandleFunc("POST /api/invoices/{id}/pdf", s.generateInvoicePDF)
	mux.HandleFunc("GET /api/invoices/{id}/pdf-file", s.serveInvoicePDF)

	// Customers
	mux.HandleFunc("GET /api/customers", s.listCustomers)
	mux.HandleFunc("POST /api/customers", s.createCustomer)
	mux.HandleFunc("GET /api/customers/{id}", s.getCustomer)
	mux.HandleFunc("PUT /api/customers/{id}", s.updateCustomer)
	mux.HandleFunc("DELETE /api/customers/{id}", s.deleteCustomer)
	mux.HandleFunc("GET /api/customers/{id}/items", s.getCustomerItems)

	// Suppliers
	mux.HandleFunc("GET /api/suppliers", s.listSuppliers)
	mux.HandleFunc("POST /api/suppliers", s.createSupplier)
	mux.HandleFunc("GET /api/suppliers/{id}", s.getSupplier)
	mux.HandleFunc("PUT /api/suppliers/{id}", s.updateSupplier)
	mux.HandleFunc("DELETE /api/suppliers/{id}", s.deleteSupplier)
	mux.HandleFunc("POST /api/suppliers/{id}/logo", s.uploadLogo)
	mux.HandleFunc("GET /api/suppliers/{id}/logo", s.serveLogo)
	mux.HandleFunc("DELETE /api/suppliers/{id}/logo", s.deleteLogo)

	// Bank accounts
	mux.HandleFunc("GET /api/suppliers/{id}/bank-accounts", s.listBankAccounts)
	mux.HandleFunc("POST /api/suppliers/{id}/bank-accounts", s.createBankAccount)
	mux.HandleFunc("PUT /api/bank-accounts/{id}", s.updateBankAccount)
	mux.HandleFunc("DELETE /api/bank-accounts/{id}", s.deleteBankAccount)

	// SMTP config (nested under suppliers) — specific routes first
	mux.HandleFunc("POST /api/suppliers/{id}/smtp/test", s.testSmtpConnection)
	mux.HandleFunc("POST /api/suppliers/{id}/smtp/copy/{fromId}", s.copySmtpConfig)
	mux.HandleFunc("GET /api/suppliers/{id}/smtp", s.getSmtpConfig)
	mux.HandleFunc("PUT /api/suppliers/{id}/smtp", s.upsertSmtpConfig)
	mux.HandleFunc("DELETE /api/suppliers/{id}/smtp", s.deleteSmtpConfig)

	// Email sending
	mux.HandleFunc("GET /api/invoices/{id}/email-preview", s.getEmailPreview)
	mux.HandleFunc("POST /api/invoices/{id}/send-email", s.sendInvoiceEmail)

	// Items catalog
	mux.HandleFunc("GET /api/items", s.listItems)
	mux.HandleFunc("POST /api/items", s.createItem)
	mux.HandleFunc("GET /api/items/most-used", s.getMostUsedItems)
	mux.HandleFunc("GET /api/items/categories", s.getItemCategories)
	mux.HandleFunc("GET /api/items/{id}", s.getItem)
	mux.HandleFunc("PUT /api/items/{id}", s.updateItem)
	mux.HandleFunc("DELETE /api/items/{id}", s.deleteItem)

	// Templates
	mux.HandleFunc("GET /api/templates", s.listTemplates)
	mux.HandleFunc("GET /api/templates/ai-prompt", s.getAIPrompt)
	mux.HandleFunc("POST /api/templates/preview-all", s.generateAllPreviews)
	mux.HandleFunc("GET /api/templates/{id}", s.getTemplate)
	mux.HandleFunc("PUT /api/templates/{id}", s.updateTemplate)
	mux.HandleFunc("DELETE /api/templates/{id}", s.deleteTemplate)
	mux.HandleFunc("PUT /api/templates/{id}/default", s.setDefaultTemplate)
	mux.HandleFunc("POST /api/templates/{id}/duplicate", s.duplicateTemplate)
	mux.HandleFunc("GET /api/templates/{id}/source", s.getTemplateSource)
	mux.HandleFunc("PUT /api/templates/{id}/source", s.updateTemplateSource)
	mux.HandleFunc("POST /api/templates/{id}/preview", s.generateTemplatePreview)
	mux.HandleFunc("GET /api/templates/{id}/preview-pdf", s.servePreviewPDF)

	// System
	mux.HandleFunc("GET /api/system/first-run", s.getFirstRun)
	mux.HandleFunc("GET /api/system/locale", s.getLocale)
	mux.HandleFunc("GET /api/system/about", s.getAbout)

	// Settings
	mux.HandleFunc("GET /api/settings", s.getSettings)
	mux.HandleFunc("PUT /api/settings", s.updateSettings)

	// Units
	mux.HandleFunc("GET /api/units", s.getUnits)
	mux.HandleFunc("PUT /api/units", s.updateUnits)

	// Payment Types
	mux.HandleFunc("GET /api/payment-types", s.getPaymentTypes)
	mux.HandleFunc("PUT /api/payment-types", s.updatePaymentTypes)

	// VAT Rates
	mux.HandleFunc("GET /api/vat-rates", s.getVATRates)
	mux.HandleFunc("PUT /api/vat-rates", s.updateVATRates)

	// Due Days Options
	mux.HandleFunc("GET /api/due-days", s.getDueDaysOptions)
	mux.HandleFunc("PUT /api/due-days", s.updateDueDaysOptions)

	// Currencies
	mux.HandleFunc("GET /api/currencies", s.getCurrencies)
	mux.HandleFunc("PUT /api/currencies", s.updateCurrencies)

	return corsMiddleware(mux)
}

var allowedOrigins = map[string]bool{
	"http://localhost:5173":   true,
	"tauri://localhost":       true,
	"https://tauri.localhost": true,
	"http://tauri.localhost":  true,
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
