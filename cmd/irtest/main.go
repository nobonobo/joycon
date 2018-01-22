package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/nobonobo/joycon"
)

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	devices, err := joycon.Search(joycon.JoyConR)
	if err != nil {
		log.Fatalln(err)
	}
	jc, err := joycon.NewJoycon(devices[0].Path, true)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("found:", jc.Name())
	defer jc.Close()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-sig:
			log.Println("signal received")
			return
		case <-t.C:
			log.Println(jc.Stats())
		case _, ok := <-jc.State():
			if !ok {
				return
			}
		case _, ok := <-jc.Sensor():
			if !ok {
				return
			}
		case ir, ok := <-jc.IRData():
			if !ok {
				return
			}
			_ = ir
		}
	}
}
