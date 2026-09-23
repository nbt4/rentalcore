package jobstatus

import "testing"

func TestCanonicalLifecycle(t *testing.T) {
	if PlanningID == ConfirmedID || ConfirmedID == CompletedID || CompletedID == CancelledID {
		t.Fatal("canonical job status IDs must be unique")
	}
	if !IsClosed(Completed) || !IsClosed(Cancelled) {
		t.Fatal("terminal statuses must be closed")
	}
	if IsClosed(Planning) || IsClosed(Confirmed) {
		t.Fatal("open statuses must not be closed")
	}
	if !IsDispatchable(Confirmed) || IsDispatchable(Planning) {
		t.Fatal("only confirmed jobs may be dispatched")
	}
}

func TestValidateTransition(t *testing.T) {
	for _, tc := range []struct {
		from, to uint
		allowed  bool
	}{
		{PlanningID, ConfirmedID, true},
		{PlanningID, CancelledID, true},
		{PlanningID, CompletedID, false},
		{ConfirmedID, CompletedID, true},
		{CompletedID, PlanningID, true},
		{CancelledID, ConfirmedID, false},
		{99, PlanningID, false},
	} {
		if got := ValidateTransition(tc.from, tc.to); (got == nil) != tc.allowed {
			t.Fatalf("transition %d -> %d: error %v, allowed %v", tc.from, tc.to, got, tc.allowed)
		}
	}
}
