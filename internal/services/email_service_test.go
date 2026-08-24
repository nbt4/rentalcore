package services

import "testing"

func TestPublicEmailAssetURL(t *testing.T) {
	t.Setenv("APP_BASE_URL", "https://rent.example.test/")
	relative := "/logos/company_print.png?v=2"
	if got := publicEmailAssetURL(&relative); got != "https://rent.example.test/logos/company_print.png?v=2" {
		t.Fatalf("unexpected public logo URL: %q", got)
	}
	absolute := "https://cdn.example.test/company.svg"
	if got := publicEmailAssetURL(&absolute); got != absolute {
		t.Fatalf("absolute URL changed: %q", got)
	}
}
