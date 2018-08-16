package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/rs/xid"
)

var (
	Guid           = xid.New()
	ProxcliDir     = ""
	GuidProxcliDir = ""
	SequenceId     = 0
)

type RequestResponse struct {
	Id       int
	TempDir  string
	Request  http.Request
	Response http.Response
}

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

func NewRequestResponse() *RequestResponse {
	SequenceId++
	r := new(RequestResponse)
	r.Id = SequenceId
	r.TempDir = fmt.Sprintf("%s/%020d", GuidProxcliDir, SequenceId)
	if err := os.Mkdir(r.TempDir, 0755); err != nil {
		panic(err)
	}

	return r
}

func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := &http.Client{}
		requestResponse := NewRequestResponse()
		requestResponse.Request = *r

		if err := requestResponse.RequestStep(); err != nil {
			panic(err)
		}
		// RequestStep
		newRequest := RequestStep(requestResponse.Request)
		bbody, err := ioutil.ReadAll(newRequest.Body)
		if err != nil {
			log.Print(err)
		}
		sbody := string(bbody)

		// Proxy
		req, err := http.NewRequest(newRequest.Method, fmt.Sprintf("http://localhost:9999%s", newRequest.URL.Path), strings.NewReader(sbody))
		if err != nil {
			log.Print(err)
		}

		res, err := client.Do(req)
		if err != nil {
			log.Print(err)
		}
		defer res.Body.Close()

		// ResponseStep
		requestResponse.Response = *res
		newResponse := ResponseStep(requestResponse.Response)
		rbody, err := ioutil.ReadAll(newResponse.Body)
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

func (r *RequestResponse) RequestStep() error {
	file := filepath.Join(r.TempDir, "Request.pcl")
	_, err := os.Stat(file)
	if err == nil {
		return err
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	log.Print(f)

	return nil
}

func RequestStep(r http.Request) *http.Request {
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

	r.Body = &ClosingBuffer{bytes.NewBufferString(sbody)}
	r.Body.Close()
	return &r
}

func ResponseStep(r http.Response) *http.Response {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Print(err)
	}
	sbody := string(body)
	log.Print(sbody)
	sbody = "change me"

	r.Body = &ClosingBuffer{bytes.NewBufferString(sbody)}
	r.Body.Close()
	return &r
}

func load() {

}

func init() {
	userinfo, err := user.Current()
	if err != nil {
		panic(err)
	}
	ProxcliDir = fmt.Sprintf("%s/.proxcli", userinfo.HomeDir)
	GuidProxcliDir = fmt.Sprintf("%s/tmp/%s", ProxcliDir, Guid.String())
	fmt.Println()
	if err := os.MkdirAll(fmt.Sprintf("%s/tmp/%s", ProxcliDir, Guid.String()), 0755); err != nil {
		panic(err)
	}
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
