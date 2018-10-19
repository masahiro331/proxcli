package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"

	"github.com/masahiro331/proxcli/proxy"
	"github.com/rs/xid"
)

var (
	Guid           = xid.New()
	ProxcliDir     = ""
	Keyfile        = ""
	Certfile       = ""
	GuidProxcliDir = ""
)

func run(sslmitm bool, port int) error {
	p := proxy.NewProxy(sslmitm, port, Certfile, Keyfile)
	return http.ListenAndServe(fmt.Sprintf("localhost:%d", port), p)
}

func main() {
	if err := run(true, 8888); err != nil {
		log.Print(err)
	}
}

func init() {
	userinfo, err := user.Current()
	if err != nil {
		panic(err)
	}
	ProxcliDir = fmt.Sprintf("%s/.proxcli", userinfo.HomeDir)
	if err := os.Mkdir(ProxcliDir, 0755); err != nil {
		log.Println(err)
	}
	if err := os.Mkdir(fmt.Sprintf("%s/ssl", ProxcliDir), 0755); err != nil {
		log.Println(err)
	}
	GuidProxcliDir = fmt.Sprintf("%s/tmp/%s", ProxcliDir, Guid.String())
	if err := os.MkdirAll(fmt.Sprintf("%s/tmp/%s", ProxcliDir, Guid.String()), 0755); err != nil {
		panic(err)
	}
	Keyfile = ProxcliDir + "/ssl/key.pem"
	Certfile = ProxcliDir + "/ssl/cert.pem"
}

// import (
// 	"bytes"
// 	"crypto/rand"
// 	"crypto/rsa"
// 	"crypto/x509"
// 	"crypto/x509/pkix"
// 	"encoding/pem"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"math/big"
// 	"net/http"
// 	"os"
// 	"os/exec"
// 	"os/user"
// 	"path/filepath"
// 	"strings"
// 	"time"
//
// 	"github.com/rs/xid"
// )
//
// var (
// 	Guid           = xid.New()
// 	ProxcliDir     = ""
// 	Keyfile        = ""
// 	Certfile       = ""
// 	GuidProxcliDir = ""
// 	SequenceId     = 0
// )
//
// type RequestResponse struct {
// 	Id       int
// 	TempDir  string
// 	Request  http.Request
// 	Response http.Response
// }
//
//
// type ClosingBuffer struct {
// 	*bytes.Buffer
// }
//
// func (cb *ClosingBuffer) Close() (err error) { return }
//
// func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {}
//
//
// func NewRequestResponse() *RequestResponse {
// 	SequenceId++
// 	r := new(RequestResponse)
// 	r.Id = SequenceId
// 	r.TempDir = fmt.Sprintf("%s/%020d", GuidProxcliDir, SequenceId)
// 	if err := os.Mkdir(r.TempDir, 0755); err != nil {
// 		panic(err)
// 	}
//
// 	return r
// }
//
// func (p *Proxy) Handler() http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		client := &http.Client{}
// 		requestResponse := NewRequestResponse()
// 		requestResponse.Request = *r
//
// 		// RequestStep
// 		if err := requestResponse.RequestStep(); err != nil {
// 			panic(err)
// 		}
// 		newRequest := RequestStep(requestResponse.Request)
// 		bbody, err := ioutil.ReadAll(newRequest.Body)
// 		if err != nil {
// 			log.Print(err)
// 		}
// 		sbody := string(bbody)
//
// 		// Proxy
// 		req, err := http.NewRequest(newRequest.Method, fmt.Sprintf("http://localhost:9999%s", newRequest.URL.Path), strings.NewReader(sbody))
// 		if err != nil {
// 			log.Print(err)
// 		}
//
// 		res, err := client.Do(req)
// 		if err != nil {
// 			log.Print(err)
// 		}
// 		defer res.Body.Close()
//
// 		// ResponseStep
// 		requestResponse.Response = *res
// 		if err := requestResponse.ResponseStep(); err != nil {
// 			panic(err)
// 		}
// 		newResponse := ResponseStep(requestResponse.Response)
// 		rbody, err := ioutil.ReadAll(newResponse.Body)
// 		if err != nil {
// 			log.Print(err)
// 		}
//
// 		// Proxy
// 		fmt.Fprintln(w, string(rbody))
// 		p.ServeHTTP(w, r)
// 	})
// }
//
// func dropCR(data []byte) []byte {
// 	if len(data) > 0 && data[len(data)-1] == '\r' {
// 		return data[0 : len(data)-1]
// 	}
// 	return data
// }
//
// func (r *RequestResponse) ResponseStep() error {
// 	file := filepath.Join(r.TempDir, "Response.pcl")
// 	_, err := os.Stat(file)
// 	if err == nil {
// 		// New.error("") みたいな処理を書かないと意味がない
// 		return err
// 	}
//
// 	f, err := r.NewResponseFile(file)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()
//
// 	// if err := runcmd("vim", file); err != nil {
// 	// 	return err
// 	// }
// 	body, err := ioutil.ReadFile(file)
// 	if err != nil {
// 		return err
// 	}
// 	r.Response.Body = &ClosingBuffer{bytes.NewBufferString(string(body))}
//
// 	return nil
// }
//
// func (r *RequestResponse) RequestStep() error {
// 	file := filepath.Join(r.TempDir, "Request.pcl")
// 	_, err := os.Stat(file)
// 	if err == nil {
// 		// New.error("") みたいな処理を書かないと意味がない
// 		return err
// 	}
//
// 	f, err := r.NewRequestFile(file)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()
//
// 	// if err := runcmd("vim", file); err != nil {
// 	// 	return err
// 	// }
//
// 	body, err := ioutil.ReadFile(file)
// 	if err != nil {
// 		return err
// 	}
// 	r.Request.Body = &ClosingBuffer{bytes.NewBufferString(string(body))}
//
// 	return nil
// }
//
// func (r *RequestResponse) NewRequestFile(file string) (*os.File, error) {
// 	f, err := os.Create(file)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()
// 	body, err := ioutil.ReadAll(r.Request.Body)
// 	if err != nil {
// 		return nil, err
// 	}
// 	f.Write(body)
//
// 	return f, nil
// }
//
// func (r *RequestResponse) NewResponseFile(file string) (*os.File, error) {
// 	f, err := os.Create(file)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()
// 	body, err := ioutil.ReadAll(r.Response.Body)
// 	if err != nil {
// 		return nil, err
// 	}
// 	f.Write(body)
//
// 	return f, nil
// }
//
// func RequestStep(r http.Request) *http.Request {
// 	body, err := ioutil.ReadAll(r.Body)
// 	if err != nil {
// 		log.Print(err)
// 	}
// 	sbody := string(body)
//
// 	r.Body = &ClosingBuffer{bytes.NewBufferString(sbody)}
// 	r.Body.Close()
// 	return &r
// }
//
// func ResponseStep(r http.Response) *http.Response {
// 	body, err := ioutil.ReadAll(r.Body)
// 	if err != nil {
// 		log.Print(err)
// 	}
// 	sbody := string(body)
// 	log.Print(sbody)
//
// 	r.Body = &ClosingBuffer{bytes.NewBufferString(sbody)}
// 	r.Body.Close()
// 	return &r
// }
// func runcmd(command, arg string) error {
// 	var cmd *exec.Cmd
// 	command += " " + arg
// 	cmd = exec.Command("sh", "-c", command)
//
// 	cmd.Stderr = os.Stderr
// 	cmd.Stdout = os.Stdout
// 	cmd.Stdin = os.Stdin
// 	return cmd.Run()
// }

// func GenerateCrt() error {
// 	Keyfile = ProxcliDir + "/ssl/key.pem"
// 	Certfile = ProxcliDir + "/ssl/cert.pem"
//
// 	_, err := os.Stat(Keyfile)
// 	if err == nil {
// 		return nil
// 	}
// 	_, err = os.Stat(Certfile)
// 	if err == nil {
// 		return nil
// 	}
//
// 	// Generate key.pem
// 	f, err := os.Create(Keyfile)
// 	if err != nil {
// 		panic(err)
// 	}
// 	f.Close()
// 	key, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		return err
// 	}
// 	keyPem := pem.EncodeToMemory(&pem.Block{
// 		Type:  "RSA PRIVATE KEY",
// 		Bytes: x509.MarshalPKCS1PrivateKey(key),
// 	})
//
// 	if err := ioutil.WriteFile(Keyfile, keyPem, 0755); err != nil {
// 		panic(err)
// 	}
//
// 	// Generate cert.pem
// 	f, err = os.Create(Certfile)
// 	if err != nil {
// 		panic(err)
// 	}
// 	f.Close()
// 	tml := x509.Certificate{
// 		NotBefore:    time.Now(),
// 		NotAfter:     time.Now().AddDate(5, 0, 0),
// 		SerialNumber: big.NewInt(000000),
// 		Subject: pkix.Name{
// 			CommonName:   "Proxcli",
// 			Organization: []string{"Example."},
// 		},
// 		BasicConstraintsValid: true,
// 	}
// 	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &key.PublicKey, key)
// 	if err != nil {
// 		log.Fatal("Certificate cannot be created.", err.Error())
// 	}
// 	certPem := pem.EncodeToMemory(&pem.Block{
// 		Type:  "CERTIFICATE",
// 		Bytes: cert,
// 	})
//
// 	if err := ioutil.WriteFile(Certfile, certPem, 0755); err != nil {
// 		panic(err)
// 	}
//
// 	if err != nil {
// 		log.Fatal("Cannot be loaded the certificate.", err.Error())
// 	}
// 	return nil
// }
//
// func main() {
// 	if err := run(); err != nil {
// 		log.Fatalf("[FATAL] %+v", err)
// 	}
// }
//
// func run() error {
// 	//ctx := context.Background()
// 	if err := GenerateCrt(); err != nil {
// 		panic(err)
// 	}
//
// 	log.Print("Start Proxy Server... \nPort 8888 listening")
// 	proxy := NewProxy(":8888", "")
// 	// return http.ListenAndServeTLS(proxy.port, Certfile, Keyfile, proxy.Handler())
// 	return http.ListenAndServe(proxy.port, proxy.Handler())
// }
