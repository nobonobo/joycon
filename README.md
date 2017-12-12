# joycon

Nintendo Switch's Joycon Device access library(via bluetooth only)

## dependencies

- github.com/flynn/hid
- github.com/shibukawa/gotomation (optional)

## usage

```go
import "github.com/nobonobo/joycon"

func main() {
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
    s := <-jc.State()
    fmt.Println(s.Buttons)  // Button bits
    fmt.Println(s.LeftAdj)  // Left Analog Stick State
    fmt.Println(s.RightAdj) // Right Analog Stick State
}
```
