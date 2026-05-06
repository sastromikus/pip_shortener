package a

import (
	"log"
	"os"
)

func Bad() {
	log.Fatal("no")
	log.Fatalf("%s", "x")
	log.Fatalln("x")
	os.Exit(1)
	panic("boom")
}