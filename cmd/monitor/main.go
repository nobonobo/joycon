package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/nobonobo/joycon"
	"gopkg.in/cheggaaa/pb.v1"
)

func calc(v float32) int {
	r := 500 - int(500*v)
	if r < 0 {
		r = 0
	}
	if r > 999 {
		r = 999
	}
	return r
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	devices, err := joycon.Search()
	if err != nil {
		log.Fatalln(err)
	}
	jcs := []*joycon.Joycon{}
	for _, dev := range devices {
		jc, err := joycon.NewJoycon(dev.Path, false)
		if err != nil {
			log.Fatalln(err)
		}
		jcs = append(jcs, jc)
	}
	defer func() {
		for _, jc := range jcs {
			jc.Close()
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	done := make(chan struct{})
	go func() {
		<-sig
		close(done)
	}()
	names := []string{
		"lsx", "lsy",
		"lax", "lay", "laz", "lgx", "lgy", "lgz",
		"rsx", "rsy",
		"rax", "ray", "raz", "rgx", "rgy", "rgz",
	}
	bars := map[string]*pb.ProgressBar{}
	barlist := []*pb.ProgressBar{}
	for _, name := range names {
		bar := pb.New(1000).Prefix(name)
		bar.ShowCounters = false
		bar.ShowTimeLeft = false
		bars[name] = bar
		barlist = append(barlist, bar)
	}
	pool, err := pb.StartPool(barlist...)
	if err != nil {
		log.Fatalln(err)
	}
	defer pool.Stop()

	var wg sync.WaitGroup
	for _, jc := range jcs {
		wg.Add(1)
		go func(jc *joycon.Joycon) {
			defer wg.Done()
			states := []joycon.State{}
			sensors := []joycon.Sensor{}
			tick := time.NewTicker(50 * time.Millisecond)
			for {
				select {
				case <-tick.C:
					//log.Printf("%s states:%d, sensors:%d", jc.Name(), len(states), len(sensors))
					if len(states) > 0 {
						last := states[len(states)-1]
						if jc.IsLeft() || jc.IsProCon() {
							bars["lsx"].Set(calc(last.LeftAdj.X))
							bars["lsy"].Set(calc(last.LeftAdj.Y))
						}
						if jc.IsRight() || jc.IsProCon() {
							bars["rsx"].Set(calc(last.RightAdj.X))
							bars["rsy"].Set(calc(last.RightAdj.Y))
						}
						states = states[0:0]
					}
					if len(sensors) > 0 {
						last := sensors[len(sensors)-1]
						if jc.IsLeft() || jc.IsProCon() {
							bars["lax"].Set(calc(last.Accel.X))
							bars["lay"].Set(calc(last.Accel.Y))
							bars["laz"].Set(calc(last.Accel.Z))
							bars["lgx"].Set(calc(last.Gyro.X))
							bars["lgy"].Set(calc(last.Gyro.Y))
							bars["lgz"].Set(calc(last.Gyro.Z))
						}
						if jc.IsRight() || jc.IsProCon() {
							bars["rax"].Set(calc(last.Accel.X))
							bars["ray"].Set(calc(last.Accel.Y))
							bars["raz"].Set(calc(last.Accel.Z))
							bars["rgx"].Set(calc(last.Gyro.X))
							bars["rgy"].Set(calc(last.Gyro.Y))
							bars["rgz"].Set(calc(last.Gyro.Z))
						}
						sensors = sensors[0:0]
					}
					continue
				case <-done:
					return
				case s, ok := <-jc.State():
					if !ok {
						return
					}
					states = append(states, s)
					/*
						log.Printf("state: %s %3d:%3d%% %06X %v%v",
							jc.Name(),
							s.Tick, s.Battery, s.Buttons, s.LeftAdj, s.RightAdj,
						)
					*/
				case s, ok := <-jc.Sensor():
					if !ok {
						return
					}
					sensors = append(sensors, s)
					/*
						log.Printf("sensor: %s %3d:%v%v",
							jc.Name(),
							s.Tick, s.Accel, s.Gyro,
						)
					*/
				}
			}
		}(jc)
	}
	wg.Wait()
}
