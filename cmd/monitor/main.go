package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/nobonobo/joycon"
)

func main() {
	log.SetFlags(log.Lmicroseconds)
	devices, err := joycon.Search()
	if err != nil {
		log.Fatalln(err)
	}
	if len(devices) == 0 {
		log.Fatalln("joycon not found")
	}
	jc, err := joycon.NewJoycon(devices[0].Path)
	if err != nil {
		log.Fatalln(err)
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	states := []joycon.State{}
	sensors := []joycon.Sensor{}
	tick := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-tick.C:
			log.Printf("states:%d, sensors:%d", len(states), len(sensors))
			states = states[0:0]
			sensors = sensors[0:0]
			continue
		case <-sig:
			jc.Close()
		case s, ok := <-jc.State():
			if !ok {
				return
			}
			states = append(states, s)
		case s, ok := <-jc.Sensor():
			if !ok {
				return
			}
			sensors = append(sensors, s)
		}
		if len(states) > 0 && len(sensors) > 0 {
			st := states[len(states)-1]
			ss := sensors[len(sensors)-1]
			log.Printf("%3d b:%3d%% btn:%06X l:%v r:%v / %3d a:%v g:%v",
				st.Tick, st.Battery, st.Buttons, st.LeftAdj, st.RightAdj,
				ss.Tick, ss.Accel, ss.Gyro)
		}
	}
}
