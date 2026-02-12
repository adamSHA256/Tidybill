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
		pdf:          service.NewPDFService(cfg.PDFDir),
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
	mux.HandleFunc("POST /api/invoices/{id}/pdf", s.generateInvoicePDF)

	// Customers
	mux.HandleFunc("GET /api/customers", s.listCustomers)
	mux.HandleFunc("POST /api/customers", s.createCustomer)
	mux.HandleFunc("GET /api/customers/{id}", s.getCustomer)
	mux.HandleFunc("PUT /api/customers/{id}", s.updateCustomer)
	mux.HandleFunc("DELETE /api/customers/{id}", s.deleteCustomer)

	// Suppliers
	mux.HandleFunc("GET /api/suppliers", s.listSuppliers)
	mux.HandleFunc("POST /api/suppliers", s.createSupplier)
	mux.HandleFunc("GET /api/suppliers/{id}", s.getSupplier)
	mux.HandleFunc("PUT /api/suppliers/{id}", s.updateSupplier)
	mux.HandleFunc("DELETE /api/suppliers/{id}", s.deleteSupplier)

	// Bank accounts
	mux.HandleFunc("GET /api/suppliers/{id}/bank-accounts", s.listBankAccounts)
	mux.HandleFunc("POST /api/suppliers/{id}/bank-accounts", s.createBankAccount)

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
