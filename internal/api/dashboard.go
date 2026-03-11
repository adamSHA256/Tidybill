package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/adamSHA256/tidybill/internal/model"
)

type CurrencyAmount struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

type DashboardStats struct {
	TotalRevenueMonth  float64          `json:"total_revenue_month"`
	RevenueByCurrency  []CurrencyAmount `json:"revenue_by_currency"`
	UnpaidCount        int              `json:"unpaid_count"`
	UnpaidAmount       float64          `json:"unpaid_amount"`
	UnpaidByCurrency   []CurrencyAmount `json:"unpaid_by_currency"`
	OverdueCount       int              `json:"overdue_count"`
	ActiveCustomers    int              `json:"active_customers"`
	InvoicesThisMonth  int              `json:"invoices_this_month"`
}

func (s *Server) getDashboardStats(w http.ResponseWriter, r *http.Request) {
	// Auto-mark overdue invoices on every dashboard load
	s.invoices.MarkOverdue()

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

	revByCur := make(map[string]float64)
	unpaidByCur := make(map[string]float64)

	for _, inv := range allInvoices {
		// Revenue this month (paid invoices)
		if inv.Status == model.StatusPaid && !inv.IssueDate.Before(monthStart) {
			stats.TotalRevenueMonth += inv.Total
			revByCur[inv.Currency] += inv.Total
		}
		// Invoices this month
		if !inv.IssueDate.Before(monthStart) {
			stats.InvoicesThisMonth++
		}
		// Unpaid amount
		if inv.Status != model.StatusPaid && inv.Status != model.StatusCancelled {
			stats.UnpaidAmount += inv.Total
			unpaidByCur[inv.Currency] += inv.Total
		}
	}

	for cur, amt := range revByCur {
		stats.RevenueByCurrency = append(stats.RevenueByCurrency, CurrencyAmount{Currency: cur, Amount: amt})
	}
	sort.Slice(stats.RevenueByCurrency, func(i, j int) bool {
		return stats.RevenueByCurrency[i].Currency < stats.RevenueByCurrency[j].Currency
	})
	for cur, amt := range unpaidByCur {
		stats.UnpaidByCurrency = append(stats.UnpaidByCurrency, CurrencyAmount{Currency: cur, Amount: amt})
	}
	sort.Slice(stats.UnpaidByCurrency, func(i, j int) bool {
		return stats.UnpaidByCurrency[i].Currency < stats.UnpaidByCurrency[j].Currency
	})

	writeJSON(w, http.StatusOK, stats)
}
