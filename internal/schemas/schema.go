package schemas

import (
	_ "embed"
)

//go:embed schema.sql
var Schema []byte
