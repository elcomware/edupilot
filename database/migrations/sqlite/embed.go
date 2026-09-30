// Package sqlitemigrations embeds the released SQLite migrations so the
// application carries its own schema (docs/adr/ADR-003). Released migrations
// are immutable: a new change is a new file.
package sqlitemigrations

import "embed"

//go:embed *.sql
var FS embed.FS
