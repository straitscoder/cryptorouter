package main

import (
	"log"
)

func main() {
	if err := Fili(); err != nil {
		log.Fatal(err)
	}
}
