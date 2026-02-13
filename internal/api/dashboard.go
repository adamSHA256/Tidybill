package api

import (
	"net/http"
	"time"

	"github.com/adamSHA256/tidybill/internal/model"
)

type DashboardStats struct {
	TotalRevenueMonth float64 `json:"total_revenue_month"`
	UnpaidCount       int     `json:"unpaid_count"`
	UnpaidAmount      float64 `json:"unpaid_amount"`
	OverdueCount      int     `json:"overdue_count"`
	ActiveCustomers   int     `json:"active_customers"`
	InvoicesThisMonth int     `json:"invoices_this_month"`
}

func (s *Server) getDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats := DashboardStats{}

	// Unpaid count
	unpaidCount, err := s.invoices.CountUnpaid()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	stats.UnpaidCount = unpaidCount

	// Overdue count
	overdueCount, err := s.invoices.CountOverdue()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	stats.OverdueCount = overdueCount

	// Active customers
	customerCount, err := s.customers.Count()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	stats.ActiveCustomers = customerCount

	// Calculate totals from invoice list
	allInvoices, err := s.invoices.List("", "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	for _, inv := range allInvoices {
		// Revenue this month (paid invoices)
		if inv.Status == model.StatusPaid && inv.IssueDate.After(monthStart) {
			stats.TotalRevenueMonth += inv.Total
		}
		// Invoices this month
		if inv.IssueDate.After(monthStart) {
			stats.InvoicesThisMonth++
		}
		// Unpaid amount
		if inv.Status != model.StatusPaid && inv.Status != model.StatusCancelled {
			stats.UnpaidAmount += inv.Total
		}
	}

	writeJSON(w, http.StatusOK, stats)
}
