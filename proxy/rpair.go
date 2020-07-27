package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
)

type RequestResponsePair struct {
	Id       int
	TempDir  string
	Request  http.Request
	Response http.Response
}

var (
	SequenceId = 0
)

func NewRequestResponsePair() *RequestResponsePair {
	SequenceId++
	r := new(RequestResponsePair)
	r.Id = SequenceId
	// r.TempDir = fmt.Sprintf("%s/%020d", GuidProxcliDir, SequenceId)
	// if err := os.Mkdir(r.TempDir, 0755); err != nil {
	// 	panic(err)
	// }

	return r
}

func (rpair *RequestResponsePair) SetRequest(r http.Request) { rpair.Request = r }

func (rpair *RequestResponsePair) SetResponse(r http.Response) { rpair.Response = r }

func (rpair *RequestResponsePair) DumpRequest() (string, error) {
	if &rpair.Request == nil {
		return "", errors.New("Request is Null")
	}
	fmt.Printf("-> Requestlog %s %s\n", rpair.Request.Method, rpair.Request.URL)
	dump, _ := httputil.DumpRequestOut(&rpair.Request, true)

	return string(dump), nil
}

func (rpair *RequestResponsePair) DumpResponse() (string, error) {
	if &rpair.Response == nil {
		return "", errors.New("Response is Null")
	}
	fmt.Printf("<- Responselog %s %s\n", rpair.Request.Method, rpair.Request.URL)
	dump, _ := httputil.DumpResponse(&rpair.Response, true)

	return string(dump), nil
}

func (rpair *RequestResponsePair) GetRequestBody() (string, error) {
	if &rpair.Request == nil {
		return "", errors.New("Request is Null")
	}

}
