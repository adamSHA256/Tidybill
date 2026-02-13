package api

import (
	"net/http"

	"github.com/adamSHA256/tidybill/internal/model"
)

func (s *Server) listCustomers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	var customers []*model.Customer
	var err error

	if q != "" {
		customers, err = s.customers.Search(q)
	} else {
		customers, err = s.customers.List()
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if customers == nil {
		customers = []*model.Customer{}
	}

	writeJSON(w, http.StatusOK, customers)
}

func (s *Server) getCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	customer, err := s.customers.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if customer == nil {
		writeError(w, http.StatusNotFound, "customer not found")
		return
	}

	writeJSON(w, http.StatusOK, customer)
}

func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request) {
	customer := model.NewCustomer()
	if err := readJSON(r, customer); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if customer.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := s.customers.Create(customer); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, customer)
}

func (s *Server) updateCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.customers.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "customer not found")
		return
	}

	if err := readJSON(r, existing); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	existing.ID = id

	if err := s.customers.Update(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) deleteCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.customers.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
