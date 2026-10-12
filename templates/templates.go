package templates

import "embed"

// Files contains the server-rendered application templates.
//
//go:embed admin/*.html auth/*.html layouts/*.html products/*.html profile/*.html
var Files embed.FS
