package jobstatus

import "strings"

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
