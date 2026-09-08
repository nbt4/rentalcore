package handlers

import (
	"database/sql"
	"net/http"
	"testing"

	"go-barcode-webapp/internal/models"
)

func TestDecodeExtractionMetadataKeepsStringFields(t *testing.T) {
	metadata := sql.NullString{
		Valid:  true,
		String: `{"title":"Luther Theater","item_count":25,"warnings":[]}`,
	}

	decoded := decodeExtractionMetadata(metadata)

	if decoded["title"] != "Luther Theater" {
		t.Fatalf("title = %q, want %q", decoded["title"], "Luther Theater")
	}
	if _, exists := decoded["item_count"]; exists {
		t.Fatal("numeric metadata should not be coerced into string metadata")
	}
}

func TestExtractionProcessingResponse(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		wantHTTP      int
		wantHandled   bool
		wantErrorText bool
	}{
		{name: "pending remains pollable", status: "pending", wantHTTP: http.StatusAccepted, wantHandled: true},
		{name: "processing remains pollable", status: "processing", wantHTTP: http.StatusAccepted, wantHandled: true},
		{name: "failed is terminal", status: "failed", wantHTTP: http.StatusUnprocessableEntity, wantHandled: true, wantErrorText: true},
		{name: "completed loads extraction", status: "completed", wantHandled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHTTP, body, handled := extractionProcessingResponse(models.PDFUpload{ProcessingStatus: tt.status})
			if handled != tt.wantHandled {
				t.Fatalf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if gotHTTP != tt.wantHTTP {
				t.Fatalf("HTTP status = %d, want %d", gotHTTP, tt.wantHTTP)
			}
			if tt.wantErrorText && body["error"] == nil {
				t.Fatal("terminal failure response has no error text")
			}
		})
	}
}
