package database

import (
	"strings"
)

// IsUniqueViolation reports whether an error is a unique-constraint failure.
//
// EduPilot supports two drivers (modernc SQLite, pgx for the cloud) and
// neither exposes a portable error type, so this matches on the driver's own
// message. The consequence is a friendly "already exists" message instead of a
// raw constraint dump; it is never used to decide anything other than
// presentation.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") ||
		strings.Contains(message, "duplicate key value") ||
		strings.Contains(message, "unique violation") ||
		strings.Contains(message, "constraint failed: unique")
}

// IsForeignKeyViolation reports whether an error is a foreign-key failure,
// which in EduPilot always means a caller referenced another school's record.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "foreign key constraint") ||
		strings.Contains(message, "violates foreign key")
}
