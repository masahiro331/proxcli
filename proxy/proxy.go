package proxy

import (
	"bufio"
	"crypto"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"

	crand "crypto/rand"

	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"sort"
	"time"
)

type Proxy struct {
	sslmitm   bool
	transport *http.Transport
	port      int
	signingCertificate
}

var certCache = map[string]*tls.Certificate{}

type signingCertificate struct {
	certificate *x509.Certificate
	privateKey  crypto.PrivateKey
}

func NewProxy(sslmitm bool, port int, Certfile, Keyfile string) *Proxy {
	p := &Proxy{
		sslmitm: sslmitm,
		transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		port: port,
	}
	p.setupCert(Certfile, Keyfile)
	return p
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		if p.sslmitm {
			p.MitmRequest(w, r)
		} else {
			p.RelayHTTPSRequest(w, r)
		}
		return
	}

	p.ProxyHTTPRequest(w, r)
}

func (p *Proxy) ProxyHTTPRequest(w http.ResponseWriter, r *http.Request) {
	rpair := NewRequestResponsePair()
	rpair.SetRequest(*r)

	dump, _ := rpair.DumpRequest()
	fmt.Println(string(dump))

	res, err := p.transport.RoundTrip(&rpair.Request)
	if err != nil {
		if res == nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	rpair.SetResponse(*res)

	dump, _ = rpair.DumpResponse()
	fmt.Println(string(dump))
	p.WriteResponse(w, &rpair.Response)

	return
}

func (p *Proxy) MitmRequest(w http.ResponseWriter, r *http.Request) {
	conn := hijackConnect(w)
	conn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))

	// launch goroutine to transporting request with mitm sniffing
	go p.ProxyHTTPSRequest(w, r, conn)
}

func (p *Proxy) ProxyHTTPSRequest(w http.ResponseWriter, r *http.Request, conn net.Conn) {
	h := r.Host
	tlsConfig, err := p.generateTLSConfig(h)
	if err != nil {
		if _, err := conn.Write([]byte("HTTP/1.0 500 Internal Server Error\r\n\r\n")); err != nil {
			log.Print(err)
		}
		conn.Close()
	}

	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		log.Print(err)
		return
	}
	// defer tlsConn.Close()

	tlsInput := bufio.NewReader(tlsConn)

	for !isEOF(tlsInput) {
		rpair := NewRequestResponsePair()
		req, err := http.ReadRequest(tlsInput)
		if err != nil {
			if err == io.EOF {
				log.Print(err)
			} else {
				log.Print(err)
			}
			return
		}
		defer req.Body.Close()

		req.URL.Scheme = "https"
		req.URL.Host = r.Host
		req.RequestURI = req.URL.String()
		req.RemoteAddr = r.RemoteAddr

		rpair.SetRequest(*req)
		dump, _ := rpair.DumpRequest()
		fmt.Println(string(dump))
		defer rpair.Request.Body.Close()

		res, err := p.transport.RoundTrip(&rpair.Request)
		if err != nil {
			fmt.Printf("error read response %v %v", r.URL.Host, err.Error())
			if res == nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		defer res.Body.Close()

		rpair.SetResponse(*res)

		dump, _ = rpair.DumpResponse()
		fmt.Println(string(dump))
		defer rpair.Response.Body.Close()

		rpair.Response.Write(tlsConn)

	}
	fmt.Printf("transportHTTPSRequest : finished ")
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

func (p *Proxy) setupCert(certfile string, keyfile string) {
	ca, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		log.Fatalf("could not load key pair: %v", err)
	}

	x509ca, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		log.Fatalf("Invalid certificate : %v", err)
	}

	p.signingCertificate = signingCertificate{
		certificate: x509ca,
		privateKey:  ca.PrivateKey,
	}
}

func (p *Proxy) findOrCreateCert(host string) (*tls.Certificate, error) {

	cert := certCache[host]
	if cert != nil {
		return cert, nil
	}

	cert, err := p.signHostCert([]string{host})
	if err == nil {
		certCache[host] = cert
	}

	return cert, err
}

func (p *Proxy) signHostCert(hosts []string) (*tls.Certificate, error) {
	now := time.Now()

	sortedHosts := make([]string, len(hosts))
	copy(sortedHosts, hosts)
	sort.Strings(sortedHosts)

	start := now.Add(-time.Minute)
	end := now.Add(30 * 3600 * time.Hour)

	h := sha1.New()
	for _, host := range sortedHosts {
		h.Write([]byte(host))
	}
	binary.Write(h, binary.BigEndian, start)
	binary.Write(h, binary.BigEndian, end)
	hash := h.Sum(nil)
	serial := big.Int{}
	serial.SetBytes(hash)

	ca := p.signingCertificate
	x509ca := ca.certificate

	template := x509.Certificate{
		SignatureAlgorithm: x509ca.SignatureAlgorithm,
		SerialNumber:       &serial,
		Issuer:             x509ca.Subject,
		Subject: pkix.Name{
			Organization: []string{"Masahiro"},
			CommonName:   hosts[0],
		},
		NotBefore:             start,
		NotAfter:              end,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:           false,
		MaxPathLen:     0,
		MaxPathLenZero: true,
		DNSNames:       hosts,
	}

	derBytes, err := x509.CreateCertificate(crand.Reader, &template, x509ca, x509ca.PublicKey, ca.privateKey)
	if err != nil {
		return nil, err
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{derBytes, x509ca.Raw},
		PrivateKey:  ca.privateKey,
	}
	return cert, nil
}

func (p *Proxy) generateTLSConfig(host string) (*tls.Config, error) {
	config := tls.Config{InsecureSkipVerify: true}

	host, _ = p.splitHostPort(host)
	cert, err := p.findOrCreateCert(host)
	if err != nil {
		return nil, err
	}

	config.Certificates = append(config.Certificates, *cert)
	return &config, nil
}

func (p *Proxy) splitHostPort(s string) (string, string) {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		port = ""
	}
	return host, port
}

func isEOF(r *bufio.Reader) bool {
	_, err := r.Peek(1)
	if err == io.EOF {
		return true
	}
	return false
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
