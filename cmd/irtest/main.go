package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/nobonobo/joycon"
)

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	devices, err := joycon.Search()
	if err != nil {
		log.Fatalln(err)
	}
	if len(devices) == 0 {
		log.Fatalln("joycon not found")
	}
	var jc *joycon.Joycon
	for _, dev := range devices {
		log.Println("found:", dev.Product)
		if dev.Product != "Joy-Con (R)" {
			continue
		}
		j, err := joycon.NewJoycon(dev.Path)
		if err != nil {
			log.Fatalln(err)
		}
		jc = j
		break
	}
	if jc == nil {
		log.Fatalln("not found supported device")
	}
	defer jc.Close()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	for {
		select {
		case <-sig:
			return
		case _, ok := <-jc.State():
			if !ok {
				return
			}
		case _, ok := <-jc.Sensor():
			if !ok {
				return
			}
		}
	}
}
