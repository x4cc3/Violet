//go:build !js || !wasm

package main

// RedirectToMdate is a no-op on non-WASM builds
func RedirectToMdate() {
	// No-op on native builds
}
