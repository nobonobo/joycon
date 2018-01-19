package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"

	"github.com/nobonobo/joycon"
)

const (
	LeftPower       = 115
	RightPower      = 127
	KEY_LEFT   uint = 65361
	KEY_UP     uint = 65362
	KEY_RIGHT  uint = 65363
	KEY_DOWN   uint = 65364
)

var (
	vibrationForLeft = []joycon.RumbleSet{
		/*
			{ // neutral
				{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // HiCoil
				{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // LoCoil
			},
		*/
		{
			{HiFreq: 64, HiAmp: LeftPower, LoFreq: 64, LoAmp: LeftPower}, // HiCoil
			{HiFreq: 64, HiAmp: LeftPower, LoFreq: 64, LoAmp: LeftPower}, // LoCoil
		},
		{
			{HiFreq: 64, HiAmp: LeftPower, LoFreq: 64, LoAmp: LeftPower}, // HiCoil
			{HiFreq: 64, HiAmp: LeftPower, LoFreq: 65, LoAmp: LeftPower}, // LoCoil
		},
		{
			{HiFreq: 64, HiAmp: LeftPower, LoFreq: 64, LoAmp: LeftPower}, // HiCoil
			{HiFreq: 64, HiAmp: LeftPower, LoFreq: 65, LoAmp: LeftPower}, // LoCoil
		},
		{
			{HiFreq: 64, HiAmp: LeftPower, LoFreq: 64, LoAmp: LeftPower}, // HiCoil
			{HiFreq: 64, HiAmp: LeftPower, LoFreq: 65, LoAmp: LeftPower}, // LoCoil
		},
	}
	vibrationForRight = []joycon.RumbleSet{
		/*
			{ // neutral
				{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // HiCoil
				{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // LoCoil
			},
		*/
		{
			{HiFreq: 64, HiAmp: RightPower, LoFreq: 64, LoAmp: RightPower}, // HiCoil
			{HiFreq: 64, HiAmp: RightPower, LoFreq: 64, LoAmp: RightPower}, // LoCoil
		},
		{
			{HiFreq: 64, HiAmp: RightPower, LoFreq: 64, LoAmp: RightPower}, // HiCoil
			{HiFreq: 64, HiAmp: RightPower, LoFreq: 65, LoAmp: RightPower}, // LoCoil
		},
		{
			{HiFreq: 64, HiAmp: RightPower, LoFreq: 64, LoAmp: RightPower}, // HiCoil
			{HiFreq: 64, HiAmp: RightPower, LoFreq: 65, LoAmp: RightPower}, // LoCoil
		},
		{
			{HiFreq: 64, HiAmp: RightPower, LoFreq: 64, LoAmp: RightPower}, // HiCoil
			{HiFreq: 64, HiAmp: RightPower, LoFreq: 65, LoAmp: RightPower}, // LoCoil
		},
	}
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
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	log.SetFlags(log.Lmicroseconds)
	devices, err := joycon.Search()
	if err != nil {
		log.Fatalln(err)
	}
	if len(devices) == 0 {
		log.Fatalln("joycon not found")
	}
	jcs := []*joycon.Joycon{}
	for _, dev := range devices {
		jc, err := joycon.NewJoycon(dev.Path)
		if err != nil {
			log.Fatalln(err)
		}
		jcs = append(jcs, jc)
	}
	if jcs[0].IsRight() {
		jcs[0], jcs[1] = jcs[1], jcs[0]
	}
	defer func() {
		for _, jc := range jcs {
			jc.Close()
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	done := make(chan struct{})
	once := sync.Once{}
	exit := func() {
		once.Do(func() { close(done) })
	}
	go func() {
		<-sig
		exit()
	}()
	go func() {
		var wg sync.WaitGroup
		for _, jc := range jcs {
			wg.Add(1)
			go func(jc *joycon.Joycon) {
				defer wg.Done()
				for {
					select {
					case <-done:
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
			}(jc)
		}
		wg.Wait()
	}()
	defer exit()

	gtk.Init(nil)
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}
	win.SetTitle("sample")
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	lbtn, _ := gtk.ButtonNew()
	lbtn.SetLabel("left")
	rbtn, _ := gtk.ButtonNew()
	rbtn.SetLabel("right")
	box.PackStart(lbtn, true, true, 0)
	box.PackStart(rbtn, true, true, 0)
	win.Add(box)

	leftOn, rightOn := false, false
	win.Connect("key-press-event", func(win *gtk.Window, ev *gdk.Event) {
		keyEvent := &gdk.EventKey{ev}
		switch keyEvent.KeyVal() {
		case KEY_LEFT:
			leftOn = true
		case KEY_RIGHT:
			rightOn = true
		}
	})
	win.Connect("key-release-event", func(win *gtk.Window, ev *gdk.Event) {
		keyEvent := &gdk.EventKey{ev}
		switch keyEvent.KeyVal() {
		case KEY_LEFT:
			leftOn = false
		case KEY_RIGHT:
			rightOn = false
		}
	})

	lbtn.Connect("pressed", func() {
		log.Println("left pressed")
		leftOn = true
	})
	lbtn.Connect("released", func() {
		log.Println("left released")
		leftOn = false
	})
	rbtn.Connect("pressed", func() {
		log.Println("right pressed")
		rightOn = true
	})
	rbtn.Connect("released", func() {
		log.Println("right released")
		rightOn = false
	})
	go func() {
		t := time.NewTicker(5 * time.Millisecond)
		for {
			select {
			case <-done:
				return
			case <-t.C:
				if leftOn {
					jcs[0].SendRumble(vibrationForLeft...)
				}
				if rightOn {
					jcs[1].SendRumble(vibrationForRight...)
				}
			}
		}
	}()

	win.SetDefaultSize(320, 200)
	win.Connect("destroy", gtk.MainQuit)
	win.Present()
	win.ShowAll()
	gtk.Main()
}
