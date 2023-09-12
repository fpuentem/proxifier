package main

import (
	"net"
	"net/http"

	redis "gopkg.in/redis.v5"

	"github.com/rookmoot/proxifier/forward"
	"github.com/rookmoot/proxifier/logger"
	"github.com/rookmoot/proxifier/proxy"
)

const (
	PROXY_PATH = "./proxy_data.json"
)

var (
	log = logger.ColorLogger{}
)

type SimpleHandler struct {
	M *proxy.Manager
}

func (t *SimpleHandler) handleRequest(conn net.Conn) {
	log.Info("New client connected")

	fwd, err := forward.New(conn, log)
	if err != nil {
		log.Warn("%v", err)
		return
	}
	defer fwd.Close()

	fwd.OnSelectRemote(func(req *http.Request) (forward.Remote, error) {
		return t.M.GetProxy()
	})

	err = fwd.Forward()
	if err != nil {
		log.Warn("%v", err)
	}
}

func main() {
	log.Verbose = true
	log.Color = true

	r := redis.NewClient(
		&redis.Options{
			Network:  "unix",
			Addr:     "/var/run/redis/redis.sock",
			Password: "",
			DB:       0,
		},
	)

	proxyManager, err := proxy.NewManager(r, log)

	if err != nil {
		panic(err)
	}

	proxyManager.UpdateProxies(PROXY_PATH)

	t := SimpleHandler{
		M: proxyManager,
	}

	addr, err := net.ResolveTCPAddr("tcp", "localhost:8080")
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Warn("%v", err)
		}

		go t.handleRequest(conn)
	}
}
