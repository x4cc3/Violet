//go:build js && wasm

package main

import (
	"embed"
	"io/fs"
)

// Embed assets
//
//go:embed all:assets
var embeddedAssets embed.FS

// Get embedded FS
// WASM only
func GetEmbeddedFS() fs.FS {
	return embeddedAssets
}

// True for WASM
func IsEmbedded() bool {
	return true
}
