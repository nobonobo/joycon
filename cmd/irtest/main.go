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
	jc, err := joycon.NewJoycon(devices[0].Path)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("found:", jc.Name())
	defer jc.Close()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	time.Sleep(500 * time.Millisecond)
	rep, err := jc.Subcommand([]byte{0x03, 0x02})
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("ir report mode0: %X", rep)
	rep, err = jc.Subcommand([]byte{0x03, 0x31})
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("change ir-mode: %X", rep)
	rep, err = jc.Subcommand([]byte{0x11})
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("send 11?: %X", rep)
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-sig:
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
