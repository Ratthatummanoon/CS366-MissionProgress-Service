package statemachine

// validTransitions defines the allowed status transitions.
var validTransitions = map[string][]string{
	"DISPATCHED":  {"EN_ROUTE"},
	"EN_ROUTE":    {"ON_SITE"},
	"ON_SITE":     {"NEED_BACKUP", "RESOLVED"},
	"NEED_BACKUP": {"ON_SITE", "RESOLVED"},
}

// validStatuses is the set of all valid statuses.
var validStatuses = map[string]bool{
	"DISPATCHED":  true,
	"EN_ROUTE":    true,
	"ON_SITE":     true,
	"NEED_BACKUP": true,
	"RESOLVED":    true,
}

// IsValidTransition checks whether transitioning from one status to another is allowed.
func IsValidTransition(from, to string) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// ValidateStatus checks if a status string is a valid enum value.
func ValidateStatus(status string) bool {
	return validStatuses[status]
}
