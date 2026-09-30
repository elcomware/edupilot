// Package postgresmigrations embeds the released PostgreSQL migrations used by
// the EduPilot Site Server and EduPilot Cloud. The logical change set mirrors
// the SQLite migrations under the same sequence numbers.
package postgresmigrations

import "embed"

//go:embed *.sql
var FS embed.FS
