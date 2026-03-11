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

	// First supplier is auto-default, subsequent ones are not
	count, err := s.suppliers.Count()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if count == 0 {
		supplier.IsDefault = true
	} else {
		supplier.IsDefault = false
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

	wasDefault := existing.IsDefault

	if err := readJSON(r, existing); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	existing.ID = id

	// Enforce single default: when setting a new default, clear others
	if existing.IsDefault && !wasDefault {
		if err := s.suppliers.SetDefault(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Prevent unsetting the current default (must set another as default instead)
	if wasDefault && !existing.IsDefault {
		existing.IsDefault = true
	}

	if err := s.suppliers.Update(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) deleteSupplier(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	invCount, err := s.invoices.CountBySupplier(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if invCount > 0 {
		writeError(w, http.StatusConflict, fmt.Sprintf("supplier has %d invoice(s) and cannot be deleted", invCount))
		return
	}

	// Check if this is the default supplier before deleting
	supplier, err := s.suppliers.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	wasDefault := supplier != nil && supplier.IsDefault

	if err := s.suppliers.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If deleted supplier was default, reassign to first remaining
	if wasDefault {
		remaining, err := s.suppliers.List()
		if err == nil && len(remaining) > 0 {
			s.suppliers.SetDefault(remaining[0].ID)
		}
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

	// First bank account for a supplier is auto-default
	baCount, err := s.bankAccounts.CountBySupplier(supplierID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if baCount == 0 {
		ba.IsDefault = true
	}

	// Clear existing defaults before creating
	if ba.IsDefault {
		if err := s.bankAccounts.ClearDefaults(supplierID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
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

	wasDefault := existing.IsDefault

	if err := readJSON(r, existing); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	existing.ID = id

	if existing.QRType != "" && !validQRTypes[existing.QRType] {
		writeError(w, http.StatusBadRequest, "invalid qr_type, must be one of: spayd, pay_by_square, epc, none")
		return
	}

	// Clear existing defaults when setting a new default
	if existing.IsDefault && !wasDefault {
		s.bankAccounts.ClearDefaults(existing.SupplierID)
	}

	// Prevent unsetting the current default (must set another as default instead)
	if wasDefault && !existing.IsDefault {
		existing.IsDefault = true
	}

	if err := s.bankAccounts.Update(existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) deleteBankAccount(w http.ResponseWriter, r *http.Request) {
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

	count, err := s.bankAccounts.CountBySupplier(existing.SupplierID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if count <= 1 {
		writeError(w, http.StatusConflict, "cannot delete the last bank account")
		return
	}

	invCount, err := s.invoices.CountByBankAccount(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if invCount > 0 {
		writeError(w, http.StatusConflict, fmt.Sprintf("account is used by %d invoice(s)", invCount))
		return
	}

	if err := s.bankAccounts.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If deleted account was default, reassign to first remaining
	if existing.IsDefault {
		accounts, err := s.bankAccounts.GetBySupplier(existing.SupplierID)
		if err == nil && len(accounts) > 0 {
			accounts[0].IsDefault = true
			s.bankAccounts.Update(accounts[0])
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
