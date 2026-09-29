//go:build !(js && wasm)

// quarkwasm is only meaningful as a WebAssembly binary; this stub keeps "go build ./..." green on
// native platforms and points the reader at the real build command.
package main

import "fmt"

func main() {
	fmt.Println("quarkwasm is a WebAssembly entry point (the engine behind the web playground).")
	fmt.Println("build it with: GOOS=js GOARCH=wasm go build -o quark.wasm ./cmd/quarkwasm")
}
