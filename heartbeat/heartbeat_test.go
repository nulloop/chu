package heartbeat_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nulloop/chu/heartbeat"
)

func TestHeartBeat(t *testing.T) {
	wait, tick := heartbeat.New(3 * time.Second)

	go func() {
		for i := 0; i < 10; i++ {
			tick()
			fmt.Println("tick")
			time.Sleep(time.Duration(i) * time.Second)
		}
		fmt.Println("done")
	}()

	wait()
	fmt.Println("too slow")

	time.Sleep(6 * time.Second)
}
