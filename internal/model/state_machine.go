package model

import "fmt"

var validTransitions = map[string][]string{
	"pending":   {"funded", "cancelled", "frozen"},
	"funded":    {"shipped", "cancelled", "disputed", "frozen"},
	"shipped":   {"delivered", "disputed", "frozen"},
	"delivered": {"completed", "disputed", "frozen"},
	"disputed":  {"completed", "cancelled", "frozen"},
	"frozen":    {"cancelled", "completed", "disputed"},
	"completed": {},
	"cancelled": {},
}

func ValidateTransition(currentStatus, nextStatus string) error {
	allowedStates, ok := validTransitions[currentStatus]
	if !ok {
		return fmt.Errorf("invalid current status: %s", currentStatus)
	}

	for _, allowed := range allowedStates {
		if allowed == nextStatus {
			return nil // Valid transition
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", currentStatus, nextStatus)
}
