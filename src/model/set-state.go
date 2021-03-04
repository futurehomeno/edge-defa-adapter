package model

// Status status
type Status struct {
}

// SetState sets correct status
func SetStatus(status string) string {
	if status == "PREPARING" {
		status = "ready_to_charge"
	}
	if status == "AVAILABLE" {
		status = "disconnected"
	}
	if status == "OCCUPIED" || status == "CHARGING" {
		status = "charging"
	}
	if status == "FINISHING" {
		status = "finished"
	}
	if status == "RESERVED" {
		status = "reserved"
	}
	if status == "UNAVAILABLE" {
		status = "unavailable"
	}
	if status == "SUSPENDEDEVSE" || status == "SUSPENDEDEV" || status == "FAULTED" {
		status = "error"
	}
	return status
}
