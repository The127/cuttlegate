// Package migrations embeds the SQL migration files so that the migration
// binary is self-contained and does not depend on the filesystem at runtime.
package migrations

import "embed"

// FS holds all files under db/migrations/.
//
//go:embed migrations/*.sql
var FS embed.FS
