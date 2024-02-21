package iperf3

import (
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

type Stat struct {
	Type     string
	Interval string
	Transfer uint64
	Bitrate  float64
	Retr     uint64 `json:",omitempty"`

	Peak float64 `json:",omitempty"`
}

type Result struct {
	Success bool
	Stats   []Stat `json:",omitempty"`
	Total   Stat   `json:",omitempty"`
	Err     string `json:",omitempty"`
}

const iperf3path = "/usr/bin/iperf3"
const timeout = 5000

func toStat(str string) Stat {
	t := strings.Fields(str[5:])
	// log.Println(str, t)
	Transfer, _ := strconv.ParseFloat(t[2], 10)
	var transfer uint64
	switch t[3] {
	case "TBytes":
		transfer = uint64(Transfer) * 1024 * 1024 * 1024 * 1024
	case "GBytes":
		transfer = uint64(Transfer) * 1024 * 1024 * 1024
	case "MBytes":
		transfer = uint64(Transfer) * 1024 * 1024
	case "KBytes":
		transfer = uint64(Transfer) * 1024
	}
	bitrate, _ := strconv.ParseFloat(t[4], 10)
	stat := Stat{
		Interval: t[0],
		Transfer: transfer,
		Bitrate:  bitrate,
	}
	if len(t) > 7 {
		retr, _ := strconv.Atoi(t[6])
		stat.Retr = uint64(retr)
	}
	// log.Println(str,stat)
	return stat
}

func analyze(stdout io.Reader, multi bool, ws *websocket.Conn) (res Result) {
	waitID := true
	buf := make([]byte, 2048)
	for {
		n, err := stdout.Read(buf)
		if err != nil {
			break
		}
		str := string(buf[:n])
		total := false
		for _, L := range strings.Split(str, "\n") {
			l := strings.TrimSpace(L)
			if l == "" {
				continue
			}
			if strings.HasPrefix(l, "iperf3: error - ") {
				res.Success = false
				res.Err = l[16:]
				if ws != nil {
					ws.WriteJSON(res)
					ws.Close()
				}
				return
			}
			if waitID {
				if strings.HasPrefix(l, "[ ID]") {
					waitID = false
				}
				continue
			}
			if strings.HasPrefix(l, "- -") {
				total = true
				waitID = true
				continue
			}
			if l[0] != '[' || (multi && l[1] != 'S') {
				continue
			}
			if strings.HasSuffix(l, "Mbits/sec") || strings.HasSuffix(l, "Bytes") {
				stat := toStat(l)
				stat.Type = "interval"
				res.Stats = append(res.Stats, stat)
				if ws != nil {
					ws.WriteJSON(stat)
				}
			}
			if total && strings.HasSuffix(l, "sender") {
				stat := toStat(l)
				stat.Type = "total"
				res.Total = stat
			}
		}
	}
	res.Success = true
	for _, stat := range res.Stats {
		if stat.Bitrate > res.Total.Peak {
			res.Total.Peak = stat.Bitrate
		}
	}
	if ws != nil {
		ws.WriteJSON(res)
		ws.Close()
	}
	// log.Println(res)
	return
}

func Client(host string, port int, reverse bool, ti int, parallel int, protocol string, ws *websocket.Conn) (res Result, err error) {
	Args := []string{
		iperf3path,
		"-c", host,
		"-p", strconv.Itoa(port),
		"-P", strconv.Itoa(parallel),
		"-t", strconv.Itoa(ti),
		"--forceflush",
		"--connect-timeout", strconv.Itoa(timeout),
		"-f", "mbps",
	}
	if reverse {
		// Args = append(Args, "--rcv-timeout", strconv.Itoa(timeout)) // unrecognized option '--rcv-timeout'
		Args = append(Args, "-R")
	}
	if protocol == "udp" {
		Args = append(Args, "-u")
	}
	// log.Println(Args)
	cmd := exec.Cmd{
		Path: iperf3path,
		Args: Args,
	}
	stdout, er := cmd.StdoutPipe()
	if er != nil {
		err = er
		res.Success = false
		res.Err = er.Error()
		if ws != nil {
			ws.WriteJSON(res)
			ws.Close()
		}
		return
	}
	cmd.Stderr = cmd.Stdout
	if err = cmd.Start(); err != nil {
		res.Success = false
		res.Err = err.Error()
		if ws != nil {
			ws.WriteJSON(res)
			ws.Close()
		}
		return
	}
	res = analyze(stdout, parallel > 1, ws)
	cmd.Wait()
	return
}

var Iperf3Server *exec.Cmd

func Serve(port int) error {
	if port == 0 {
		port = 5201
	}
	if Iperf3Server == nil {
		Iperf3Server = &exec.Cmd{
			Path: iperf3path,
			Args: []string{
				iperf3path,
				"-s",
				"-p", strconv.Itoa(port),
			},
		}
		if err := Iperf3Server.Start(); err != nil {
			Iperf3Server = nil
			return err
		}
	}
	return nil
}

func Shutdown() error {
	if Iperf3Server == nil {
		return nil
	}
	if err := Iperf3Server.Process.Kill(); err != nil {
		return err
	}
	Iperf3Server.Wait()
	Iperf3Server = nil
	return nil
}
