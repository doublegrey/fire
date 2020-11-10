package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/goombaio/namegenerator"
	"github.com/gorilla/websocket"
)

type counter uint64

type event struct {
	Name    string `json:"name"`
	Rps     uint64 `json:"rps"`
	Latency uint64 `json:"latency"`
	Errors  uint64 `json:"errors"`
}

var (
	rps     counter
	latency counter
	errors  counter
	master  = flag.String("master", "localhost:8080", "master server address")
	addr    = flag.String("addr", "", "pong server address")
	name    = flag.String("name", "random", "worker name")
)

func (c *counter) get() uint64 {
	return atomic.LoadUint64((*uint64)(c))
}
func (c *counter) set(value uint64) uint64 {
	atomic.StoreUint64((*uint64)(c), value)
	return value
}
func (c *counter) inc() uint64 {
	return atomic.AddUint64((*uint64)(c), 1)
}
func (c *counter) reset() uint64 {
	return atomic.SwapUint64((*uint64)(c), 0)
}

func ping() {
	if *addr == "" {
		log.Fatal("pong server address is not specified\n./worker --master localhost:8080 --addr 192.168.0.200:12345")
	}
	for {
		start := time.Now()
		resp, err := http.Get(*addr)
		if err != nil {
			errors.inc()
			continue
		}
		if resp.StatusCode != http.StatusTeapot {
			log.Printf("invalid response status: %d\n", resp.StatusCode)
			errors.inc()
			continue
		}
		l := time.Since(start)

		latency.set((latency.get() + uint64(l.Milliseconds())) / 2)
		rps.inc()
	}
}

func main() {
	flag.Parse()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	if *name == "random" {
		seed := time.Now().UTC().UnixNano()
		nameGenerator := namegenerator.NewNameGenerator(seed)
		*name = nameGenerator.Generate()
	}

	u := url.URL{Scheme: "ws", Host: *master, Path: "/events"}

	log.Printf("connecting to %s\n", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("failed to connect to ws: %s\n", err)
	}
	log.Printf("connected as: %s\n", *name)
	defer c.Close()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	go ping()
	for {
		select {
		case <-ticker.C:
			e := event{Name: *name, Rps: rps.get(), Latency: latency.get(), Errors: errors.get()}
			bytes, err := json.Marshal(e)
			if err != nil {
				log.Printf("failed to marshal event data: %s\n", err)
				continue
			}
			err = c.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				log.Printf("failed to write message: %s\n", err)
				continue
			}
			rps.reset()
		case <-interrupt:
			log.Println("closing connection...")
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			os.Exit(0)
		}
	}
}
