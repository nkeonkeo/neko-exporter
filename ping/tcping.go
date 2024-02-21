package ping

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"time"
)

func TCPing(ip string, port int, count int, interval, timeout time.Duration, verbose bool) (Result, error) {
	taddr := &net.TCPAddr{IP: net.ParseIP(ip), Port: port}
	packets := make([]packet, 0, count)
	sum, sd, min, max := float64(0), float64(0), float64(100000), float64(0)
	recv, sent := 0, 0
	recvc := make(chan packet, count)

	t := time.NewTicker(interval)
	defer t.Stop()
	if verbose {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				t.Stop()
				close(recvc)
			}
		}()
	}
	go func() {
		for i := 0; i < count; i++ {
			go func(seq int) {
				sent++
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
				defer cancel()
				st := time.Now()
				rc, err := net.DialTCPContext(ctx, "tcp", nil, taddr)
				if err != nil {
					recvc <- packet{Rtt: 0, Seq: seq, Err: err}
					return
				}
				defer rc.Close()
				recvc <- packet{Rtt: float64(time.Since(st).Microseconds()) / 1000, Seq: seq}
			}(i)
			if _, ok := <-t.C; !ok {
				break
			}
		}
	}()
	for i := 0; i < count; i++ {
		p, ok := <-recvc
		if !ok {
			break
		}
		packets = append(packets, p)
		if p.Err != nil {
			if verbose {
				fmt.Println("TCPing to", ip, "Err:", p.Err)
			}
			continue
		}
		recv++
		sum += p.Rtt
		if p.Rtt > max {
			max = p.Rtt
		}
		if p.Rtt < min {
			min = p.Rtt
		}
		if verbose {
			fmt.Printf("TCPing from %s: seq=%d time=%.2fms\n", ip, p.Seq, p.Rtt)
		}
	}
	if recv == 0 {
		return Result{
			IP:          ip,
			Sent:        sent,
			Recv:        recv,
			LossPercent: float64(100),
		}, nil
	}
	avg := sum / float64(recv)
	for _, p := range packets {
		x := p.Rtt - avg
		sd += x * x
	}
	res := Result{
		IP:          ip,
		Sent:        sent,
		Recv:        recv,
		LossPercent: float64(sent-recv) / float64(count) * 100,
		Avg:         avg,
		Min:         min,
		Max:         max,
		Stdev:       math.Sqrt(sd / float64(recv)),
		RecvPackets: packets,
	}
	if verbose {
		fmt.Printf("\n--- %s ping statistics ---\n", ip)
		fmt.Printf("%d packets transmitted, %d packets received, %.2f%% packet loss\n",
			res.Sent, res.Recv, res.LossPercent)
		fmt.Printf("rtt min/avg/max/stdev = %.2fms/%.2fms/%.2fms/%.2fms\n",
			res.Min, res.Avg, res.Max, res.Stdev)
	}
	return res, nil
}
