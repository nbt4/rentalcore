package postalcode

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLookupReturnsUniqueCitiesAndCachesResult(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.URL.Query().Get("postalCode"); got != "35708" {
			t.Fatalf("postalCode = %q, want 35708", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name":"Haiger","postalCode":"35708"},
			{"name":"Haiger","postalCode":"35708"},
			{"name":"Falscher Ort","postalCode":"99999"}
		]`))
	}))
	defer server.Close()

	client := newClient(server.URL, server.Client())
	for range 2 {
		cities, err := client.Lookup(context.Background(), "35708")
		if err != nil {
			t.Fatalf("Lookup() error = %v", err)
		}
		if len(cities) != 1 || cities[0] != "Haiger" {
			t.Fatalf("cities = %#v, want [Haiger]", cities)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestLookupRejectsInvalidPostalCode(t *testing.T) {
	client := newClient("https://example.invalid", http.DefaultClient)
	_, err := client.Lookup(context.Background(), "1234")
	if !errors.Is(err, ErrInvalidPostalCode) {
		t.Fatalf("error = %v, want ErrInvalidPostalCode", err)
	}
}
