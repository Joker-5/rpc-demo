package rpc_demo

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"rpc_demo/codec"
)

func startServer(addr chan string) {
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	Accept(l)
}
func TestServer_ServeConn(t *testing.T) {
	addr := make(chan string)
	go startServer(addr)

	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	_ = json.NewEncoder(conn).Encode(DefaultOption)
	cc := codec.NewJsonCodec(conn)

	Convey("Server#ServeConn", t, func() {
		Convey("Test01", func() {
			for i := 0; i < 5; i++ {
				h := &codec.Header{
					ServiceMethod: "Foo.Sum",
					Seq:           uint64(i),
				}
				_ = cc.Write(h, fmt.Sprintf("rpc req %d", h.Seq))
				_ = cc.ReadHeader(h)
				var reply string
				_ = cc.ReadBody(&reply)
				log.Println("reply:", reply)
				So(reply, ShouldNotEqual, nil)
			}
		})
	})

}
