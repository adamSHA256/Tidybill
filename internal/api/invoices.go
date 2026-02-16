package api

import (
	"net/http"
	"time"

	"github.com/adamSHA256/tidybill/internal/database/repository"
	"github.com/adamSHA256/tidybill/internal/model"
	"github.com/adamSHA256/tidybill/internal/service"
)

func (s *Server) listInvoices(w http.ResponseWriter, r *http.Request) {
	status := model.InvoiceStatus(r.URL.Query().Get("status"))
	customerID := r.URL.Query().Get("customer_id")
	supplierID := r.URL.Query().Get("supplier_id")

	invoices, err := s.invoices.List(status, customerID, supplierID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Load items, customer and supplier for each invoice
	for _, inv := range invoices {
		items, err := s.invoiceItems.GetByInvoice(inv.ID)
		if err == nil {
			inv.Items = items
		}
		cust, err := s.customers.GetByID(inv.CustomerID)
		if err == nil && cust != nil {
			inv.Customer = cust
		}
		sup, err := s.suppliers.GetByID(inv.SupplierID)
		if err == nil && sup != nil {
			inv.Supplier = sup
		}
	}

	if invoices == nil {
		invoices = []*model.Invoice{}
	}

	writeJSON(w, http.StatusOK, invoices)
}

func (s *Server) getInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	inv, err := s.invoices.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if inv == nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	// Load relations
	items, err := s.invoiceItems.GetByInvoice(inv.ID)
	if err == nil {
		inv.Items = items
	}
	cust, err := s.customers.GetByID(inv.CustomerID)
	if err == nil && cust != nil {
		inv.Customer = cust
	}
	sup, err := s.suppliers.GetByID(inv.SupplierID)
	if err == nil && sup != nil {
		inv.Supplier = sup
	}

	writeJSON(w, http.StatusOK, inv)
}

type CreateInvoiceRequest struct {
	CustomerID    string               `json:"customer_id"`
	SupplierID    string               `json:"supplier_id"`
	BankAccountID string               `json:"bank_account_id"`
	InvoiceNumber string               `json:"invoice_number"`
	IssueDate     string               `json:"issue_date"`
	DueDate       string               `json:"due_date"`
	Currency      string               `json:"currency"`
	Notes         string               `json:"notes"`
	InternalNotes string               `json:"internal_notes"`
	Items         []CreateItemRequest   `json:"items"`
}

type CreateItemRequest struct {
	ItemID      string  `json:"item_id"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"`
	VATRate     float64 `json:"vat_rate"`
}

func (s *Server) createInvoice(w http.ResponseWriter, r *http.Request) {
	var req CreateInvoiceRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.CustomerID == "" || req.SupplierID == "" {
		writeError(w, http.StatusBadRequest, "customer_id and supplier_id are required")
		return
	}

	// Get supplier for invoice number generation
	supplier, err := s.suppliers.GetByID(req.SupplierID)
	if err != nil || supplier == nil {
		writeError(w, http.StatusBadRequest, "supplier not found")
		return
	}

	// Use provided invoice number or auto-generate
	invNumber := req.InvoiceNumber
	if invNumber == "" {
		var err error
		invNumber, err = s.invoices.GetNextNumber(supplier.ID, supplier.InvoicePrefix)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	inv := model.NewInvoice(req.SupplierID, req.CustomerID, req.BankAccountID)
	inv.InvoiceNumber = invNumber
	inv.VariableSymbol = repository.GenerateVariableSymbol(invNumber)
	inv.Notes = req.Notes

	if req.Currency != "" {
		inv.Currency = req.Currency
	}

	// Parse dates if provided
	if req.IssueDate != "" {
		if t, err := time.Parse("2006-01-02", req.IssueDate); err == nil {
			inv.IssueDate = t
			inv.TaxableDate = t
		}
	}
	if req.DueDate != "" {
		if t, err := time.Parse("2006-01-02", req.DueDate); err == nil {
			inv.DueDate = t
		}
	}

	// Create invoice
	if err := s.invoices.Create(inv); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create items and calculate totals
	var subtotal, vatTotal float64
	items := make([]model.InvoiceItem, len(req.Items))
	for i, itemReq := range req.Items {
		item := model.InvoiceItem{
			InvoiceID:   inv.ID,
			ItemID:      itemReq.ItemID,
			Description: itemReq.Description,
			Quantity:    itemReq.Quantity,
			Unit:        itemReq.Unit,
			UnitPrice:   itemReq.UnitPrice,
			VATRate:     itemReq.VATRate,
			Position:    i + 1,
		}
		item.Calculate()
		subtotal += item.Subtotal
		vatTotal += item.VATAmount
		items[i] = item
	}

	// Auto-create catalog entries for items without item_id
	for i := range items {
		if items[i].ItemID == "" && items[i].Description != "" {
			existing, _ := s.items.FindByDescription(items[i].Description)
			if existing != nil {
				items[i].ItemID = existing.ID
			} else {
				catalogItem := model.NewItem()
				catalogItem.Description = items[i].Description
				catalogItem.DefaultPrice = items[i].UnitPrice
				catalogItem.DefaultUnit = items[i].Unit
				catalogItem.DefaultVATRate = items[i].VATRate
				catalogItem.LastUsedPrice = items[i].UnitPrice
				catalogItem.LastCustomerID = req.CustomerID
				if err := s.items.Create(catalogItem); err == nil {
					items[i].ItemID = catalogItem.ID
				}
			}
		}
	}

	if len(items) > 0 {
		if err := s.invoiceItems.CreateBatch(items); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Track usage for catalog items
	for _, item := range items {
		if item.ItemID != "" {
			s.items.IncrementUsage(item.ItemID, item.UnitPrice, req.CustomerID)
			s.custItems.Upsert(req.CustomerID, item.ItemID, item.UnitPrice, item.Quantity)
		}
	}

	// Update invoice totals
	inv.Subtotal = subtotal
	inv.VATTotal = vatTotal
	inv.Total = subtotal + vatTotal
	inv.Items = items
	if err := s.invoices.Update(inv); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, inv)
}

func (s *Server) updateInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.invoices.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	var req CreateInvoiceRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.CustomerID != "" {
		existing.CustomerID = req.CustomerID
	}
	if req.BankAccountID != "" {
		existing.BankAccountID = req.BankAccountID
	}
	if req.Currency != "" {
		existing.Currency = req.Currency
	}
	existing.Notes = req.Notes
	existing.InternalNotes = req.InternalNotes

	if req.IssueDate != "" {
		if t, err := time.Parse("2006-01-02", req.IssueDate); err == nil {
			existing.IssueDate = t
			existing.TaxableDate = t
		}
	}
	if req.DueDate != "" {
		if t, err := time.Parse("2006-01-02", req.DueDate); err == nil {
			existing.DueDate = t
		}
	}

	// Replace items if provided
	if len(req.Items) > 0 {
		if err := s.invoiceItems.DeleteByInvoice(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		var subtotal, vatTotal float64
		items := make([]model.InvoiceItem, len(req.Items))
		for i, itemReq := range req.Items {
			item := model.InvoiceItem{
				InvoiceID:   id,
				ItemID:      itemReq.ItemID,
				Description: itemReq.Description,
				Quantity:    itemReq.Quantity,
				Unit:        itemReq.Unit,
				UnitPrice:   itemReq.UnitPrice,
				VATRate:     itemReq.VATRate,
				Position:    i + 1,
			}
			item.Calculate()
			subtotal += item.Subtotal
			vatTotal += item.VATAmount
			items[i] = item
		}

		if err := s.invoiceItems.CreateBatch(items); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		existing.Subtotal = subtotal
		existing.VATTotal = vatTotal
		existing.Total = subtotal + vatTotal
	}

	if err := s.invoices.Update(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) deleteInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.invoiceItems.DeleteByInvoice(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.invoices.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type UpdateStatusRequest struct {
	Status model.InvoiceStatus `json:"status"`
}

func (s *Server) updateInvoiceStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateStatusRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if err := s.invoices.UpdateStatus(id, req.Status); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	inv, _ := s.invoices.GetByID(id)
	writeJSON(w, http.StatusOK, inv)
}

type UpdateNotesRequest struct {
	InternalNotes string `json:"internal_notes"`
}

func (s *Server) updateInvoiceNotes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateNotesRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if err := s.invoices.UpdateInternalNotes(id, req.InternalNotes); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	inv, _ := s.invoices.GetByID(id)
	writeJSON(w, http.StatusOK, inv)
}

func (s *Server) generateInvoicePDF(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	inv, err := s.invoices.GetByID(id)
	if err != nil || inv == nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	supplier, err := s.suppliers.GetByID(inv.SupplierID)
	if err != nil || supplier == nil {
		writeError(w, http.StatusInternalServerError, "supplier not found")
		return
	}

	customer, err := s.customers.GetByID(inv.CustomerID)
	if err != nil || customer == nil {
		writeError(w, http.StatusInternalServerError, "customer not found")
		return
	}

	bankAccount, err := s.bankAccounts.GetByID(inv.BankAccountID)
	if err != nil || bankAccount == nil {
		writeError(w, http.StatusInternalServerError, "bank account not found")
		return
	}

	items, err := s.invoiceItems.GetByInvoice(inv.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := &service.InvoiceData{
		Invoice:     inv,
		Supplier:    supplier,
		Customer:    customer,
		BankAccount: bankAccount,
		Items:       items,
	}

	// Determine template: use request body override or invoice's template_id
	templateCode := inv.TemplateID
	var req struct {
		TemplateID string `json:"template_id"`
	}
	// Ignore errors - body is optional
	readJSON(r, &req)
	if req.TemplateID != "" {
		templateCode = req.TemplateID
	}

	// Look up template settings from DB
	opts := &service.TemplateOptions{
		ShowLogo:  true,
		ShowQR:    true,
		ShowNotes: true,
		QRType:    bankAccount.QRType,
	}
	if tmpl, err := s.templates.GetByID(templateCode); err == nil && tmpl != nil {
		opts.ShowLogo = tmpl.ShowLogo
		opts.ShowQR = tmpl.ShowQR
		opts.ShowNotes = tmpl.ShowNotes
		templateCode = tmpl.TemplateCode
	}

	pdfPath, err := s.pdf.GenerateInvoice(data, templateCode, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PDF generation failed: "+err.Error())
		return
	}

	// Update invoice with PDF path
	inv.PDFPath = pdfPath
	inv.Status = model.StatusCreated
	s.invoices.Update(inv)

	writeJSON(w, http.StatusOK, map[string]string{"path": pdfPath})
}
