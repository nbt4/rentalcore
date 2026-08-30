package repository

import (
	"reflect"
	"testing"
)

func TestProductSearchTerms(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "combines brand and model", input: " LD Systems  Stinger SUB 18A G3 ", want: []string{"ld", "systems", "stinger", "sub", "18a", "g3"}},
		{name: "empty", input: "   ", want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProductSearchTerms(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ProductSearchTerms(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}
