package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-barcode-webapp/internal/services/postalcode"

	"github.com/gin-gonic/gin"
)

type postalCodeLookupStub struct {
	cities []string
	err    error
}

func (s postalCodeLookupStub) Lookup(context.Context, string) ([]string, error) {
	return s.cities, s.err
}

func TestLookupPostalCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		lookup     PostalCodeLookup
		wantStatus int
		wantCity   string
	}{
		{
			name:       "returns matching city",
			lookup:     postalCodeLookupStub{cities: []string{"Haiger"}},
			wantStatus: http.StatusOK,
			wantCity:   "Haiger",
		},
		{
			name:       "rejects invalid postal code",
			lookup:     postalCodeLookupStub{err: postalcode.ErrInvalidPostalCode},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "keeps manual entry available on upstream error",
			lookup:     postalCodeLookupStub{err: errors.New("upstream unavailable")},
			wantStatus: http.StatusBadGateway,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := &CustomerHandler{postalLookup: test.lookup}
			router := gin.New()
			router.GET("/postal-code/:postalCode", handler.LookupPostalCode)
			request := httptest.NewRequest(http.MethodGet, "/postal-code/35708", nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantCity != "" {
				var body struct {
					Cities []string `json:"cities"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if len(body.Cities) != 1 || body.Cities[0] != test.wantCity {
					t.Fatalf("cities = %#v, want [%s]", body.Cities, test.wantCity)
				}
			}
		})
	}
}
