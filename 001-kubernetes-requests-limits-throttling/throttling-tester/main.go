package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)
func init() {
	fmt.Println("todle se fakt samo pousti? ;-)")
}

func main() {
	processes, _ := strconv.Atoi(os.Getenv("PROCESSES"))
	ticks, _ := strconv.Atoi(os.Getenv("TICKS"))
	tick_length, _ := strconv.Atoi(os.Getenv("TICKLENGTH"))
	if tick_length == 0 {
		fmt.Println("WARNING !!! : You forgot to specify ticklength")
		tick_length = 1
	}
	fmt.Printf("Running throttling test for %d processes for %d ticks! \n", processes, ticks+1)
	//processes := 4
	runtime.GOMAXPROCS(processes + 1)
	for i:=0; i<processes; i++ {
		go func(){
			x := 0
			for {
				x = x + x
			}
		}()
	}
	// this creates a new ticker which will
	// `tick` every 1 second.
	ticker := time.NewTicker(time.Duration(tick_length) * time.Millisecond)
	// for every `tick` that our `ticker`
	// emits, we print `tock`
	counter := 0
	last_t := time.Time{}
	for t := range ticker.C {
		counter++
		fmt.Printf("#%d: Diff: %s, invoked at: %s \n", counter, t.Sub(last_t), t)
		last_t = t
		if counter >= ticks {
			break;
		}
	}
}