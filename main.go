package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type RequestResponse struct {
	Id       int
	Request  *http.Request
	Response *http.Response
}

// type Request struct {
// 	Method      string
// 	Scheme      string
// 	Hostname    string
// 	Port        int
// 	Path        string
// 	QueryString string
// 	Header      string
// 	Body        string
// }
// type Response struct {
// 	Status string
// 	Header string
// 	Body   string
// }
// func NewRequestResponse struct { }

type Proxy struct {
	port    string
	keyFile string
}

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() (err error) { return }

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func NewProxy(port string, keyfile string) *Proxy {
	return &Proxy{
		port:    port,
		keyFile: keyfile,
	}
}

func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := &http.Client{}
		requestResponse := new(RequestResponse)
		fmt.Printf("(pointer 1): %s\n", *r)
		requestResponse.Request = r
		fmt.Printf("(pointer 2): %s\n", *r)
		fmt.Printf("(pointer 3): %s\n", *requestResponse.Request)

		// RequestStep
		r = RequestStep(requestResponse.Request)
		fmt.Printf("(pointer 4): %s\n", *r)
		fmt.Printf("(pointer 5): %s\n", *requestResponse.Request)
		bbody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Print(err)
		}
		sbody := string(bbody)

		// Proxy
		req, err := http.NewRequest(r.Method, fmt.Sprintf("http://localhost:9999%s", r.URL.Path), strings.NewReader(sbody))
		if err != nil {
			log.Print(err)
		}

		res, err := client.Do(req)
		if err != nil {
			log.Print(err)
		}
		defer res.Body.Close()

		// ResponseStep
		res = ResponseStep(res)
		rbody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Print(err)
		}

		// Proxy
		fmt.Fprintln(w, string(rbody))
		p.ServeHTTP(w, r)
	})
}
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func RequestStep(r *http.Request) *http.Request {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Print(err)
	}
	sbody := string(body)

	// scanner := bufio.NewScanner(os.Stdin)
	// for scanner.Scan() {
	// 	text := scanner.Text()
	// 	log.Println(text)
	// }
	// log.Print(sbody)

	r.Body = &ClosingBuffer{bytes.NewBufferString(sbody)}
	r.Body.Close()
	return r
}

func ResponseStep(r *http.Response) *http.Response {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Print(err)
	}
	sbody := string(body)
	log.Print(sbody)
	sbody = "change me"

	r.Body = &ClosingBuffer{bytes.NewBufferString(sbody)}
	r.Body.Close()
	return r
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("[FATAL] %+v", err)
	}
}

func run() error {
	//ctx := context.Background()

	proxy := NewProxy(":8888", "")
	return http.ListenAndServe(proxy.port, proxy.Handler())
}
