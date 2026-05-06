package main

import (
	"log"
	"os"
)

func main() {
	log.Fatal("ok here")
	os.Exit(0)
}

func helper() {
	log.Fatal("not ok")
	os.Exit(2)
	panic("x")
}
