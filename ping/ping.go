package ping

import (
	"net"
	"time"
)

type packet struct {
	Rtt float64
	Seq int
	Err error
}

type Result struct {
	IP                   string
	Sent                 int
	Recv                 int
	LossPercent          float64
	LastPacket           packet
	Avg, Min, Max, Stdev float64
	RecvPackets          []packet
	Err                  error
}

func Ping(host string, port int, count int, interval, timeout int, protocol string, verbose bool) (Result, error) {
	var ips []string
	ips, err := net.LookupHost(host)
	if err != nil {
		return Result{}, err
	}
	if port == 0 {
		port = 22
	}
	if count == 0 {
		count = 30
	}
	if interval == 0 {
		interval = 1000
	}
	Interval := time.Duration(interval) * time.Millisecond
	if timeout == 0 {
		timeout = 1000
	}
	Timeout := time.Duration(timeout+count*interval) * time.Millisecond
	switch protocol {
	case "tcp":
		return TCPing(ips[0], port, count, Interval, Timeout, verbose)
	default:
		return ICMPing(ips[0], count, Interval, Timeout, verbose)
	}
}
