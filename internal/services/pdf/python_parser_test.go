package pdf

import "testing"

func TestParsedDocumentType(t *testing.T) {
	tests := []struct {
		input string
		want  ParsedDocumentType
	}{
		{input: "offer", want: DocTypeOffer},
		{input: "invoice", want: DocTypeInvoice},
		{input: "order", want: DocTypeOrder},
		{input: "delivery", want: DocTypeDelivery},
		{input: "", want: DocTypeUnknown},
	}

	for _, test := range tests {
		if got := parsedDocumentType(test.input); got != test.want {
			t.Fatalf("parsedDocumentType(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
