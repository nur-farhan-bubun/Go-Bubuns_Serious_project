package docs

import "embed"

//go:embed specs/*.yaml
var SpecFS embed.FS
