package constant

const (
	StatusPending   = "Pending"
	StatusConfirmed = "Confirmed"
	StatusProcess   = "Process"
	StatusSending   = "Sending"
	StatusCancelled = "Cancelled"
	StatusCompleted = "Completed"
)

var ValidOrderTransitions = map[string]map[string]bool{
	StatusPending: {
		StatusConfirmed: true,
		StatusCancelled: true,
	},
	StatusConfirmed: {
		StatusProcess:   true,
		StatusCancelled: true,
	},
	StatusProcess: {
		StatusSending: true,
	},
	StatusSending: {
		StatusCompleted: true,
	},
}

func IsValidTransition(currentStatus, nextStatus string) bool {
	allowedNextStatuses, exists := ValidOrderTransitions[currentStatus]
	if !exists {
		return false
	}
	return allowedNextStatuses[nextStatus]
}
