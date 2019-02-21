package routing

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// WebSocketReverseProxy implements http.HandlerFunc to reverse proxy websocket requests
type WebSocketReverseProxy struct {
	Target string
}

// NewWebSocketReverseProxy creates a new websocket reverse proxy
func NewWebSocketReverseProxy(url *url.URL) *WebSocketReverseProxy {
	var proxy = new(WebSocketReverseProxy)
	proxy.Target = fmt.Sprintf("%s:%s", url.Hostname(), url.Port())

	return proxy
}

func (ws *WebSocketReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d, err := net.Dial("tcp", ws.Target)
	if err != nil {
		http.Error(w, "Error contacting backend server.", http.StatusBadGateway)
		log.Printf("Error dialing websocket backend %s: %s", ws.Target, err)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Not a hijacker?", http.StatusInternalServerError)
		return
	}

	nc, _, err := hj.Hijack()
	if err != nil {
		log.Printf("Hijack error: %v", err)
		return
	}

	defer nc.Close()
	defer d.Close()

	err = r.Write(d)
	if err != nil {
		log.Printf("Error copying request to target: %v", err)
		return
	}

	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)

		if err != nil {
			errc <- err
		}
	}
	go cp(d, nc)
	go cp(nc, d)
	<-errc
}

// IsWebSocket determines whether or not an http request is using websocket
func IsWebSocket(r *http.Request) bool {
	connHdr := ""
	connHdrs := r.Header["Connection"]
	if len(connHdrs) > 0 {
		connHdr = connHdrs[0]
	}

	upgradeWs := false
	if strings.ToLower(connHdr) == "upgrade" {
		upgradeHdrs := r.Header["Upgrade"]
		if len(upgradeHdrs) > 0 {
			upgradeWs = (strings.ToLower(upgradeHdrs[0]) == "websocket")
		}
	}

	return upgradeWs
}