package api

import "net/http"

// GET /api/invoices/{id}/email-preview
func (s *Server) getEmailPreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	preview, err := s.emailService.GetEmailPreview(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, preview)
}

// POST /api/invoices/{id}/send-email
func (s *Server) sendInvoiceEmail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		To       string `json:"to"`
		Subject  string `json:"subject"`
		Body     string `json:"body"`
		SendCopy bool   `json:"send_copy"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.To == "" || req.Subject == "" {
		writeError(w, http.StatusBadRequest, "to and subject are required")
		return
	}

	if err := s.emailService.SendInvoiceEmail(id, req.To, req.Subject, req.Body, req.SendCopy); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Re-load invoice to get updated fields
	inv, _ := s.invoices.GetByID(id)
	resp := map[string]interface{}{
		"ok": true,
	}
	if inv != nil {
		resp["email_sent_at"] = inv.EmailSentAt
		resp["status"] = inv.Status
	}

	writeJSON(w, http.StatusOK, resp)
}
