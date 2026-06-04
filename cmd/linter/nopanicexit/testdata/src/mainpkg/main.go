package main

import (
	logger "log"
	"os"
)

func main() {
	logger.Fatal("ok here")
	os.Exit(0)
}

func helper() {
	logger.Fatal("not ok") // want "log.Fatal is allowed only in main.main"
	os.Exit(2)             // want "os.Exit is allowed only in main.main"
	panic("x")             // want "panic usage is forbidden"
}
