package i18n

import "testing"

func TestTranslateUnit(t *testing.T) {
	tests := []struct {
		unit string
		lang Lang
		want string
	}{
		{"ks", CS, "ks"},
		{"ks", EN, "pcs"},
		{"ks", SK, "ks"},
		{"hod", CS, "hod"},
		{"hod", EN, "hr"},
		{"hod", SK, "hod"},
		{"den", CS, "den"},
		{"den", EN, "day"},
		{"den", SK, "deň"},
		// Unknown units returned as-is
		{"m²", EN, "m²"},
		{"kg", CS, "kg"},
		{"custom_unit", EN, "custom_unit"},
	}

	for _, tt := range tests {
		got := TranslateUnit(tt.unit, tt.lang)
		if got != tt.want {
			t.Errorf("TranslateUnit(%q, %q) = %q, want %q", tt.unit, tt.lang, got, tt.want)
		}
	}
}
