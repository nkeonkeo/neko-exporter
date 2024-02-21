package ping

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/go-ping/ping"
	"github.com/gorilla/websocket"
)

func ICMPing(ip string, count int, interval, timeout time.Duration, verbose bool) (Result, error) {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return Result{IP: ip, Err: err}, err
	}
	if count > 0 {
		pinger.Count = count
	}
	pinger.Interval = interval
	pinger.Timeout = timeout
	if verbose {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				pinger.Stop()
			}
		}()
	}
	packets := make([]packet, 0, count)
	pinger.OnRecv = func(p *ping.Packet) {
		packets = append(packets, packet{
			Rtt: float64(p.Rtt.Microseconds()) / 1000,
			Seq: p.Seq,
		})
		if verbose {
			fmt.Printf("%d bytes from %s: icmp_seq=%d time=%.2fms\n",
				p.Nbytes, p.IPAddr, p.Seq, float32(p.Rtt.Microseconds())/1000.0)
		}
	}
	if verbose {
		pinger.OnDuplicateRecv = func(pkt *ping.Packet) {
			fmt.Printf("%d bytes from %s: icmp_seq=%d time=%.2f ttl=%v (DUP!)\n",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, float32(pkt.Rtt.Microseconds())/1000.0, pkt.Ttl)
		}

		pinger.OnFinish = func(stats *ping.Statistics) {
			fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
			fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
				stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
			fmt.Printf("rtt min/avg/max/stdev = %v/%v/%v/%v\n",
				stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
		}
	}
	if err := pinger.Run(); err != nil {
		return Result{IP: ip, Err: err}, err
	}
	s := pinger.Statistics()
	return Result{
		IP:          ip,
		Sent:        s.PacketsSent,
		Recv:        s.PacketsRecv,
		LossPercent: s.PacketLoss,
		Avg:         float64(s.AvgRtt.Microseconds()) / 1000,
		Min:         float64(s.MinRtt.Microseconds()) / 1000,
		Max:         float64(s.MaxRtt.Microseconds()) / 1000,
		Stdev:       float64(s.StdDevRtt.Microseconds()) / 1000,
		RecvPackets: packets,
	}, nil
}

func ICMPingWs(ip string, count int, interval, timeout time.Duration, ws *websocket.Conn) {
	defer ws.Close()
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		ws.WriteJSON(Result{IP: ip, Err: err})
		return
	}
	pinger.Count = count
	pinger.Interval = interval
	pinger.Timeout = timeout
	packets := make([]packet, 0, count)
	pinger.OnRecv = func(p *ping.Packet) {
		packet := packet{
			Rtt: float64(p.Rtt.Microseconds()) / 1000,
			Seq: p.Seq,
		}
		packets = append(packets, packet)

		stats := pinger.Statistics()
		ws.WriteJSON(Result{
			IP:          ip,
			Sent:        stats.PacketsSent,
			Recv:        stats.PacketsRecv,
			LossPercent: stats.PacketLoss,
			LastPacket:  packet,
			Avg:         float64(stats.AvgRtt.Microseconds()) / 1000,
			Min:         float64(stats.MinRtt.Microseconds()) / 1000,
			Max:         float64(stats.MaxRtt.Microseconds()) / 1000,
			Stdev:       float64(stats.StdDevRtt.Microseconds()) / 1000,
			RecvPackets: packets,
		})

		// fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n",p.Nbytes, p.IPAddr, p.Seq, p.Rtt)
	}
	pinger.OnDuplicateRecv = func(pkt *ping.Packet) {
		// fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v (DUP!)\n",pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
	}

	pinger.OnFinish = func(stats *ping.Statistics) {
		ws.WriteJSON(Result{
			IP:          ip,
			Sent:        stats.PacketsSent,
			Recv:        stats.PacketsRecv,
			LossPercent: stats.PacketLoss,
			Avg:         float64(stats.AvgRtt.Microseconds()) / 1000,
			Min:         float64(stats.MinRtt.Microseconds()) / 1000,
			Max:         float64(stats.MaxRtt.Microseconds()) / 1000,
			Stdev:       float64(stats.StdDevRtt.Microseconds()) / 1000,
			RecvPackets: packets,
		})

		// fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
		// fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
		// 	stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
		// fmt.Printf("round-trip min/avg/max/stdev = %v/%v/%v/%v\n",
		// 	stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
	}
	if err := pinger.Run(); err != nil {
		ws.WriteJSON(Result{IP: ip, Err: err})
		return
	}
}
