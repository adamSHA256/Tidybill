package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

var validQRTypes = map[string]bool{
	"spayd": true, "pay_by_square": true, "epc": true, "none": true,
}

const maxLogoSize = 2 << 20 // 2 MB

var validLogoTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
}

func (s *Server) uploadLogo(w http.ResponseWriter, r *http.Request) {
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

	r.Body = http.MaxBytesReader(w, r.Body, maxLogoSize+512) // small buffer for multipart overhead
	if err := r.ParseMultipartForm(maxLogoSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large (max 2 MB)")
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing logo file")
		return
	}
	defer file.Close()

	if header.Size > maxLogoSize {
		writeError(w, http.StatusBadRequest, "file too large (max 2 MB)")
		return
	}

	// Detect content type from first 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}
	contentType := http.DetectContentType(buf[:n])

	ext, ok := validLogoTypes[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported file type (PNG and JPG only)")
		return
	}

	// Seek back to start
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	// Delete old logo if exists
	if supplier.LogoPath != "" {
		os.Remove(supplier.LogoPath)
	}

	// Save file
	filename := fmt.Sprintf("%s%s", id, ext)
	destPath := filepath.Join(s.cfg.LogoDir, filename)

	dst, err := os.Create(destPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save logo")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(destPath)
		writeError(w, http.StatusInternalServerError, "failed to save logo")
		return
	}

	// Update supplier
	supplier.LogoPath = destPath
	if err := s.suppliers.Update(supplier); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, supplier)
}

func (s *Server) serveLogo(w http.ResponseWriter, r *http.Request) {
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

	if supplier.LogoPath == "" {
		writeError(w, http.StatusNotFound, "no logo set")
		return
	}

	// Path traversal protection
	absPath, err := filepath.Abs(supplier.LogoPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid path")
		return
	}
	absLogoDir, err := filepath.Abs(s.cfg.LogoDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid config")
		return
	}
	if !strings.HasPrefix(absPath, absLogoDir+string(os.PathSeparator)) && absPath != absLogoDir {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		supplier.LogoPath = ""
		s.suppliers.Update(supplier)
		writeError(w, http.StatusNotFound, "logo file not found")
		return
	}

	http.ServeFile(w, r, absPath)
}

func (s *Server) deleteLogo(w http.ResponseWriter, r *http.Request) {
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

	if supplier.LogoPath != "" {
		os.Remove(supplier.LogoPath)
	}

	supplier.LogoPath = ""
	if err := s.suppliers.Update(supplier); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateBankAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.bankAccounts.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "bank account not found")
		return
	}

	if err := readJSON(r, existing); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	existing.ID = id

	if existing.QRType != "" && !validQRTypes[existing.QRType] {
		writeError(w, http.StatusBadRequest, "invalid qr_type, must be one of: spayd, pay_by_square, epc, none")
		return
	}

	if err := s.bankAccounts.Update(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}
