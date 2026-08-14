package domains

import (
	"testing"
)

func TestNormalizeDomainName(t *testing.T) {
	tests := []struct {
		input     string
		expected  string
		expectErr bool
	}{
		{"Example.COM", "example.com", false},
		{"NOREST.MAIL", "norest.mail", false},
		{"already-lower.io", "already-lower.io", false},
		{"MiXeD.CaSe.DoMaIn", "mixed.case.domain", false},
		{"example.com.", "example.com", false},
		{"", "", true},
		{"A", "a", false},
		{"invalid name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := NormalizeDomainName(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("NormalizeDomainName(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizeDomainName(%q) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("NormalizeDomainName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
