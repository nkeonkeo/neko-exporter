package main

import (
	"neko-exporter/iperf3"
	"strconv"

	"github.com/gin-gonic/gin"
)

func Iperf3(c *gin.Context) {
	host := c.PostForm("host")
	port, _ := strconv.Atoi(c.PostForm("port"))
	if port == 0 {
		port = 5201
	}
	reverse := c.PostForm("reverse") == "true"
	time, _ := strconv.Atoi(c.PostForm("time"))
	if time == 0 {
		time = 5
	}
	parallel, _ := strconv.Atoi(c.PostForm("parallel"))
	if parallel == 0 {
		parallel = 1
	}
	protocol := c.PostForm("protocol")
	if protocol == "" {
		protocol = "tcp"
	}
	res, err := iperf3.Client(host, port, reverse, time, parallel, protocol, nil)
	if err == nil {
		resp(c, true, res, 200)
	} else {
		resp(c, false, err.Error(), 500)
	}
}

func Iperf3Ws(c *gin.Context) {
	host := c.Query("host")
	port, _ := strconv.Atoi(c.Query("port"))
	if port == 0 {
		port = 5201
	}
	reverse := c.Query("reverse") != ""
	time, _ := strconv.Atoi(c.Query("time"))
	if time == 0 {
		time = 10
	}
	parallel, _ := strconv.Atoi(c.Query("parallel"))
	if parallel == 0 {
		parallel = 1
	}
	protocol := c.Query("protocol")
	if protocol == "" {
		protocol = "tcp"
	}
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	iperf3.Client(host, port, reverse, time, parallel, protocol, ws)
}

func Iperf3Serve(c *gin.Context) {
	port, _ := strconv.Atoi(c.Query("port"))
	if err := iperf3.Serve(port); err == nil {
		resp(c, true, nil, 200)
	} else {
		resp(c, false, err.Error(), 500)
	}
}
func Iperf3Shutdown(c *gin.Context) {
	if err := iperf3.Shutdown(); err == nil {
		resp(c, true, nil, 200)
	} else {
		resp(c, false, err.Error(), 500)
	}
}
