package main

import (
	"fmt"
	"log"

	"github.com/nobonobo/joycon"
)

func main() {
	devices, err := joycon.Search()
	if err != nil {
		log.Fatalln(err)
	}
	for _, device := range devices {
		fmt.Printf("%s: %q\n", device.Product, device.Path)
	}
}
