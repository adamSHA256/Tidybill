package api

import (
	"net/http"

	"github.com/adamSHA256/tidybill/internal/model"
)

func (s *Server) listSuppliers(w http.ResponseWriter, r *http.Request) {
	suppliers, err := s.suppliers.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if suppliers == nil {
		suppliers = []*model.Supplier{}
	}

	writeJSON(w, http.StatusOK, suppliers)
}

func (s *Server) getSupplier(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	supplier, err := s.suppliers.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if supplier == nil {
		writeError(w, http.StatusNotFound, "supplier not found")
		return
	}

	writeJSON(w, http.StatusOK, supplier)
}

func (s *Server) createSupplier(w http.ResponseWriter, r *http.Request) {
	supplier := model.NewSupplier()
	if err := readJSON(r, supplier); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if supplier.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := s.suppliers.Create(supplier); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, supplier)
}

func (s *Server) updateSupplier(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.suppliers.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "supplier not found")
		return
	}

	if err := readJSON(r, existing); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	existing.ID = id

	if err := s.suppliers.Update(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) deleteSupplier(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.suppliers.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listBankAccounts(w http.ResponseWriter, r *http.Request) {
	supplierID := r.PathValue("id")

	accounts, err := s.bankAccounts.GetBySupplier(supplierID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if accounts == nil {
		accounts = []*model.BankAccount{}
	}

	writeJSON(w, http.StatusOK, accounts)
}

func (s *Server) createBankAccount(w http.ResponseWriter, r *http.Request) {
	supplierID := r.PathValue("id")

	ba := model.NewBankAccount(supplierID)
	if err := readJSON(r, ba); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	ba.SupplierID = supplierID

	if ba.AccountNumber == "" {
		writeError(w, http.StatusBadRequest, "account_number is required")
		return
	}

	if err := s.bankAccounts.Create(ba); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ba)
}
