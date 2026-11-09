package migrations

import _ "embed"

//go:embed 000001_reference.sql
var Schema string
