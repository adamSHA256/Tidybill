package gdrive

// E2E tests for the gdrive Transport are in gdrive_e2e_test.go
// and gated by TIDYBILL_E2E_GDRIVE=1.
// Unit-level mocking of driveapi.Service is impractical without substantial
// fixture work; the package-level compilation test is the primary coverage here.
