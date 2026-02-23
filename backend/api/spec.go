package api

import "embed"

// SpecFS embeds the OpenAPI specification file.
//
//go:embed openapi.yaml
var SpecFS embed.FS
