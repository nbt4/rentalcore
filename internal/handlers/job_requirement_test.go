package handlers

import "testing"

func TestJobRequirementCreateRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request JobRequirementCreateRequest
		wantErr bool
	}{
		{name: "valid", request: JobRequirementCreateRequest{ProductID: 42, Quantity: 3}},
		{name: "missing product", request: JobRequirementCreateRequest{Quantity: 3}, wantErr: true},
		{name: "zero quantity", request: JobRequirementCreateRequest{ProductID: 42}, wantErr: true},
		{name: "negative quantity", request: JobRequirementCreateRequest{ProductID: 42, Quantity: -1}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.request.validate(); (err != nil) != test.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestJobRequirementUpdateRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request JobRequirementUpdateRequest
		wantErr bool
	}{
		{name: "valid", request: JobRequirementUpdateRequest{Quantity: 3}},
		{name: "zero quantity", request: JobRequirementUpdateRequest{}, wantErr: true},
		{name: "negative quantity", request: JobRequirementUpdateRequest{Quantity: -1}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.request.validate(); (err != nil) != test.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
