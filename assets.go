package reta

import "embed"

//go:embed VERSION
var VersionRaw string

//go:embed all:templates
var TemplatesFS embed.FS

//go:embed lib/base.mk
var BaseMK string
