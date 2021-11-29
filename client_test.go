package rpc_demo

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestClient_Call(t *testing.T) {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)
	client, _ := Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup

	Convey("test client#call", t, func() {
		Convey("test01", func() {
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					args := fmt.Sprintf("rpc req %d", i)
					var reply string
					if err := client.Call("Foo.Sum", args, &reply); err != nil {
						log.Fatal("call Foo.Sum error:", err)
					}
					log.Println("reply:", reply)
				}(i)
			}
			wg.Wait()
		})
	})
}
