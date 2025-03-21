package prompt

import "testing"

func TestGetLanguage(t *testing.T) {
	testCases := []struct {
		langCode string
		expected string
	}{
		{"en", "English"},
		{"zh-tw", "Traditional Chinese"},
		{"zh-cn", "Simplified Chinese"},
		{"ja", "Japanese"},
		{"fr", "English"},
	}

	for _, tc := range testCases {
		result := GetLanguage(tc.langCode)
		if result != tc.expected {
			t.Errorf("GetLanguage(%q) = %q, expected %q", tc.langCode, result, tc.expected)
		}
	}
}
