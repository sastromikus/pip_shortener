package a

import (
	logger "log"
	"os"
)

// Bad is a test fixture with forbidden termination calls.
func Bad() {
	logger.Fatal("no")       // want "log.Fatal is allowed only in main.main"
	logger.Fatalf("%s", "x") // want "log.Fatal is allowed only in main.main"
	logger.Fatalln("x")      // want "log.Fatal is allowed only in main.main"
	os.Exit(1)               // want "os.Exit is allowed only in main.main"
	panic("boom")            // want "panic usage is forbidden"
}

// Shadowed is a test fixture with a local panic identifier.
func Shadowed() {
	panic := func(string) {}
	panic("not builtin")
}

// PanicHelper is a test fixture wrapping the builtin panic function.
func PanicHelper() {
	panic("wrapped") // want "panic usage is forbidden"
}

type guard struct{}

func (guard) panicIf(failed bool) {
	if failed {
		panic("method wrapper") // want "panic usage is forbidden"
	}
}

// WrappedPanicCall verifies that calls through function and method wrappers are reported.
func WrappedPanicCall() {
	PanicHelper()         // want "panic usage is forbidden"
	guard{}.panicIf(true) // want "panic usage is forbidden"
}
