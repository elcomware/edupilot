package domain

import "time"

// dateLayout is the storage format for a date-only value.
const dateLayout = "2006-01-02"

// dateOnly strips the time component. Birthdays, year boundaries and term
// boundaries must never depend on the timezone a value was typed in, or the
// same person gets two different birthdays on two EduPilot nodes.
func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
