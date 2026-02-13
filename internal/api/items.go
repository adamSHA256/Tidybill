package api

import (
	"net/http"

	"github.com/adamSHA256/tidybill/internal/model"
)

func (s *Server) listItems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	var items []*model.Item
	var err error

	if q != "" {
		items, err = s.items.Search(q)
	} else {
		items, err = s.items.List(0, 0)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if items == nil {
		items = []*model.Item{}
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	item, err := s.items.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

type CatalogItemRequest struct {
	Description    string  `json:"description"`
	DefaultPrice   float64 `json:"default_price"`
	DefaultUnit    string  `json:"default_unit"`
	DefaultVATRate float64 `json:"default_vat_rate"`
	Category       string  `json:"category"`
}

func (s *Server) createItem(w http.ResponseWriter, r *http.Request) {
	var req CatalogItemRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	// Check for duplicate
	existing, _ := s.items.FindByDescription(req.Description)
	if existing != nil {
		writeError(w, http.StatusConflict, "item with this description already exists")
		return
	}

	item := model.NewItem()
	item.Description = req.Description
	item.DefaultPrice = req.DefaultPrice
	item.DefaultUnit = req.DefaultUnit
	item.DefaultVATRate = req.DefaultVATRate
	item.Category = req.Category

	if item.DefaultUnit == "" {
		item.DefaultUnit = "ks"
	}

	if err := s.items.Create(item); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.items.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}

	var req CatalogItemRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Description != "" {
		existing.Description = req.Description
	}
	existing.DefaultPrice = req.DefaultPrice
	existing.DefaultUnit = req.DefaultUnit
	existing.DefaultVATRate = req.DefaultVATRate
	existing.Category = req.Category

	if err := s.items.Update(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.items.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getMostUsedItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.items.GetMostUsed(10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if items == nil {
		items = []*model.Item{}
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getItemCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.items.GetExistingCategories()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if categories == nil {
		categories = []string{}
	}

	writeJSON(w, http.StatusOK, categories)
}

func (s *Server) getCustomerItems(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("id")

	items, err := s.custItems.GetByCustomer(customerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if items == nil {
		items = []*model.CustomerItem{}
	}

	writeJSON(w, http.StatusOK, items)
}
