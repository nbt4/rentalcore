package jobstatus

import (
	"fmt"
	"strings"
)

const (
	PlanningID  uint = 1
	ConfirmedID uint = 2
	CompletedID uint = 4
	CancelledID uint = 6

	Planning  = "Planung"
	Confirmed = "Bestätigt"
	Completed = "Abgeschlossen"
	Cancelled = "Storniert"
)

var (
	OpenIDs     = []uint{PlanningID, ConfirmedID}
	TerminalIDs = []uint{CompletedID, CancelledID}
)

func IsClosed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "abgeschlossen", "storniert", "completed", "paid", "canceled", "cancelled", "abgerechnet":
		return true
	default:
		return false
	}
}

func IsDispatchable(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), Confirmed)
}

// ValidID reports whether a status belongs to the shared job lifecycle.
func ValidID(id uint) bool {
	switch id {
	case PlanningID, ConfirmedID, CompletedID, CancelledID:
		return true
	default:
		return false
	}
}

// ValidateTransition keeps operational state changes explicit. A finished or
// cancelled job can be reopened as a draft before it is confirmed again.
func ValidateTransition(from, to uint) error {
	if !ValidID(from) || !ValidID(to) {
		return fmt.Errorf("invalid job status")
	}
	if from == to {
		return nil
	}
	switch from {
	case PlanningID:
		if to == ConfirmedID || to == CancelledID {
			return nil
		}
	case ConfirmedID:
		if to == PlanningID || to == CompletedID || to == CancelledID {
			return nil
		}
	case CompletedID, CancelledID:
		if to == PlanningID {
			return nil
		}
	}
	return fmt.Errorf("job status transition from %d to %d is not allowed", from, to)
}
