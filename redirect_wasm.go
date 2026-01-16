//go:build js && wasm

package main

import (
	"syscall/js"
)

// RedirectToMdate redirects the browser to the mdate website
func RedirectToMdate() {
	js.Global().Get("window").Get("location").Set("href", "https://mdate.vercel.app/")
}
