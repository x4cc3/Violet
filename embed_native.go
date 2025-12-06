//go:build !js || !wasm

package main

import "io/fs"

// Nil for native
func GetEmbeddedFS() fs.FS {
	return nil
}

// False for native
func IsEmbedded() bool {
	return false
}
