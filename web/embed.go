// Package web embeds the frontend so deployment still requires one binary.
package web

import "embed"

//go:embed pages/*.html static
var Files embed.FS
