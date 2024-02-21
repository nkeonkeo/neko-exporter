package mtr

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	tm "github.com/buger/goterm"
	"github.com/gorilla/websocket"
	"github.com/tonobo/mtr/pkg/hop"
	"github.com/tonobo/mtr/pkg/icmp"
	"github.com/tonobo/mtr/pkg/mtr"
)

type packet struct {
	Success bool
	Respond float64
}

type Node struct {
	Host        string
	Sent        int
	TTL         int
	LossPercent float64
	Last        float64
	Avg         float64
	Best        float64
	Worst       float64
	Packets     []packet
}

type Result struct {
	Host      string
	Statistic []Node
	Err       string `json:",omitempty"`
}

var (
	TIMEOUT          = 800 * time.Millisecond
	INTERVAL         = 100 * time.Millisecond
	HOP_SLEEP        = time.Nanosecond
	MAX_HOPS         = 64
	MAX_UNKNOWN_HOPS = 10
	RING_BUFFER_SIZE = 50
	PTR_LOOKUP       = false
	srcAddr          = ""
)

func packets(h *hop.HopStatistic) []packet {
	v := make([]packet, 0, h.RingBufferSize)
	h.Packets.Do(func(f interface{}) {
		if f == nil {
			return
		}
		x := f.(icmp.ICMPReturn)
		if x.Success {
			v = append(v, packet{
				Success: true,
				Respond: x.Elapsed.Seconds() * 1000,
			})
		} else {
			v = append(v, packet{
				Success: false,
				Respond: 0.0,
			})
		}
	})
	return v
}

func toNode(h *hop.HopStatistic, hide bool) Node {
	node := Node{
		Host:        h.Target,
		Sent:        h.Sent,
		TTL:         h.TTL,
		LossPercent: h.Loss(),
		Last:        h.Last.Elapsed.Seconds() * 1000,
		Best:        h.Best.Elapsed.Seconds() * 1000,
		Worst:       h.Worst.Elapsed.Seconds() * 1000,
		Avg:         h.Avg(),
		Packets:     packets(h),
	}
	if hide && node.Host != "" {
		t := strings.Split(node.Host, ".")
		if len(t) >= 1 {
			t[len(t)-1] = "x"
		}
		if len(t) >= 2 {
			t[len(t)-2] = "x"
		}
		node.Host = strings.Join(t, ".")
	}
	return node
}

func toRes(m *mtr.MTR) Result {
	res := Result{
		Host:      m.Address,
		Statistic: make([]Node, 0),
	}
	for i := 1; i <= len(m.Statistic); i++ {
		res.Statistic = append(res.Statistic, toNode(m.Statistic[i], i <= 5))
	}
	return res
}

func Mtr(host string, count int, hide bool, ws *websocket.Conn) (res Result, err error) {
	ips, er := net.LookupHost(host)
	if er != nil {
		res.Err = er.Error()
		if ws != nil {
			ws.WriteJSON(res)
			ws.Close()
		}
		err = er
		return
	}
	m, ch, er := mtr.NewMTR(ips[0], srcAddr, TIMEOUT, INTERVAL, HOP_SLEEP, MAX_HOPS, MAX_UNKNOWN_HOPS, RING_BUFFER_SIZE, PTR_LOOKUP)
	if er != nil {
		err = er
		return
	}
	go func(ch chan struct{}) {
		m.Run(ch, count)
		close(ch)
	}(ch)
	for range ch {
		if ws != nil {
			ws.WriteJSON(toRes(m))
		}
	}
	if ws != nil {
		ws.Close()
	}
	res = toRes(m)
	return
}

func CMD(host string, count, interval, timeout int) {
	if interval == 0 {
		interval = 100
	}
	Interval := time.Duration(interval) * time.Millisecond
	if timeout == 0 {
		timeout = 1000
	}
	Timeout := time.Duration(timeout) * time.Millisecond
	m, ch, err := mtr.NewMTR(host, srcAddr, Timeout, Interval, HOP_SLEEP,
		MAX_HOPS, MAX_UNKNOWN_HOPS, RING_BUFFER_SIZE, PTR_LOOKUP)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Start:", time.Now())
	tm.Clear()
	mu := &sync.Mutex{}
	go func(ch chan struct{}) {
		for {
			mu.Lock()
			<-ch
			render(m)
			mu.Unlock()
		}
	}(ch)
	m.Run(ch, count)
	close(ch)
	mu.Lock()
	render(m)
	mu.Unlock()
}

func render(m *mtr.MTR) {
	tm.MoveCursor(1, 1)
	m.Render(1)
	tm.Flush() // Call it every time at the end of rendering
}
