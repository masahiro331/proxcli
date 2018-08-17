package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

type Proxy struct {
	sslmitm   bool
	transport *http.Transport
	port      int
}

func NewProxy(sslmitm bool, port int) *Proxy {
	p := &Proxy{
		sslmitm: sslmitm,
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
		port: port,
	}
	return p
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		if p.sslmitm {
			conn := hijackConnect(w)
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			p.TransportHTTPSRequest(w, r, conn)
		} else {
			p.RelayHTTPSRequest(w, r)
		}
		return
	}

	p.TransportHTTPRequest(w, r)
}

func (p *Proxy) TransportHTTPRequest(w http.ResponseWriter, r *http.Request) {
	rpair := NewRequestResponsePair()
	rpair.SetRequest(*r)

	dump, _ := rpair.DumpRequest()
	fmt.Println(dump)

	res, err := p.transport.RoundTrip(&rpair.Request)
	if err != nil {
		if res == nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	rpair.SetResponse(*res)

	dump, _ = rpair.DumpResponse()
	fmt.Println(dump)
	p.WriteResponse(w, &rpair.Response)

	return
}

func (p *Proxy) TransportHTTPSRequest(w http.ResponseWriter, r *http.Request, conn net.Conn) {
	h := r.Host
}

func ResponseConnect(w http.ResponseWriter) net.Conn {
	conn := hijackConnect(w)
	conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	return conn
}

func (p *Proxy) RelayHTTPSRequest(w http.ResponseWriter, r *http.Request) {
	dst, err := net.Dial("tcp", r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	// conn := hijackConnect(w)
	// conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	conn := ResponseConnect(w)

	go transfer(dst, conn)
	go transfer(conn, dst)
}

func (p *Proxy) WriteResponse(w http.ResponseWriter, res *http.Response) {
	dst := w.Header()
	for k, vs := range res.Header {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
	w.WriteHeader(res.StatusCode)

	_, err := io.Copy(w, res.Body)
	if err != nil {
		log.Print(err)
	}

	if err := res.Body.Close(); err != nil {
		log.Print(err)
	}
}

func hijackConnect(w http.ResponseWriter) net.Conn {
	hj, ok := w.(http.Hijacker)
	if !ok {
		panic("httpserver does not support hijacking")
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		panic("Cannot hijack connection " + err.Error())
	}
	return conn
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}
