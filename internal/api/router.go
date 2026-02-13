package api

import (
	"database/sql"
	"net/http"

	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database/repository"
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
	pdf          *service.PDFService
	cfg          *config.Config
}

func NewServer(db *sql.DB, cfg *config.Config) *Server {
	return &Server{
		invoices:     repository.NewInvoiceRepository(db),
		invoiceItems: repository.NewInvoiceItemRepository(db),
		customers:    repository.NewCustomerRepository(db),
		suppliers:    repository.NewSupplierRepository(db),
		bankAccounts: repository.NewBankAccountRepository(db),
		settings:     repository.NewSettingsRepository(db),
		items:        repository.NewItemRepository(db),
		custItems:    repository.NewCustomerItemRepository(db),
		templates:    repository.NewPDFTemplateRepository(db),
		pdf:          service.NewPDFService(cfg.PDFDir, cfg.PreviewDir),
		cfg:          cfg,
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Dashboard
	mux.HandleFunc("GET /api/dashboard/stats", s.getDashboardStats)

	// Invoices
	mux.HandleFunc("GET /api/invoices", s.listInvoices)
	mux.HandleFunc("POST /api/invoices", s.createInvoice)
	mux.HandleFunc("GET /api/invoices/{id}", s.getInvoice)
	mux.HandleFunc("PUT /api/invoices/{id}", s.updateInvoice)
	mux.HandleFunc("DELETE /api/invoices/{id}", s.deleteInvoice)
	mux.HandleFunc("PUT /api/invoices/{id}/status", s.updateInvoiceStatus)
	mux.HandleFunc("PUT /api/invoices/{id}/notes", s.updateInvoiceNotes)
	mux.HandleFunc("POST /api/invoices/{id}/pdf", s.generateInvoicePDF)

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

	// Bank accounts
	mux.HandleFunc("GET /api/suppliers/{id}/bank-accounts", s.listBankAccounts)
	mux.HandleFunc("POST /api/suppliers/{id}/bank-accounts", s.createBankAccount)
	mux.HandleFunc("PUT /api/bank-accounts/{id}", s.updateBankAccount)

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
	mux.HandleFunc("POST /api/templates/preview-all", s.generateAllPreviews)
	mux.HandleFunc("GET /api/templates/{id}", s.getTemplate)
	mux.HandleFunc("PUT /api/templates/{id}", s.updateTemplate)
	mux.HandleFunc("PUT /api/templates/{id}/default", s.setDefaultTemplate)
	mux.HandleFunc("POST /api/templates/{id}/preview", s.generateTemplatePreview)
	mux.HandleFunc("GET /api/templates/{id}/preview-pdf", s.servePreviewPDF)

	// Settings
	mux.HandleFunc("GET /api/settings", s.getSettings)
	mux.HandleFunc("PUT /api/settings", s.updateSettings)

	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
