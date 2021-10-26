package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pborman/uuid"
)

const (
	port = "8080"
)

func main() {
	fmt.Printf("Starting proxy on port: %s\n", port)
	handler := &httpHandler{}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nStopping proxy.")
		os.Exit(1)
	}()

	http.ListenAndServe(fmt.Sprintf(":%s", port), handler)
}

type httpHandler struct {
	httpCLi http.Client
}

func (h *httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	sessionID := uuid.New()

	redirectUrl := strings.Replace(r.URL.RequestURI(), "/", "", 1)
	parsedRedirectUrl, err := url.Parse(redirectUrl)

	if err != nil {
		errStr := fmt.Errorf("unable to parse URL: '%s' error: %w", redirectUrl, err)
		fmt.Println(errStr)

		_ = returnProxyError(rw, errStr.Error())
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Errorf("error reading request body: %w", err)
		fmt.Println(errStr)

		_ = returnProxyError(rw, errStr.Error())
		return
	}
	defer r.Body.Close()

	fmt.Printf(`
[REQUEST ID] : %s
[URL]: %s
[METHOD]: %s
[HEADERS]:
%s
[REQUEST BODY]:
%s

`, sessionID, redirectUrl, r.Method, headerToPrintableFormat(r.Header), string(bodyBytes))

	buffer := bytes.NewBuffer(bodyBytes)
	nopCloser := ioutil.NopCloser(buffer)

	httpReq := &http.Request{
		Method: r.Method,
		URL:    parsedRedirectUrl,
		Header: r.Header,
		Body:   nopCloser,
	}

	res, err := h.httpCLi.Do(httpReq)
	if err != nil {
		errStr := fmt.Errorf("error executing http request: %s session ID: %s", err.Error(), sessionID)
		fmt.Println(errStr)

		_ = returnProxyError(rw, errStr.Error())
		return
	}
	defer res.Body.Close()

	rw.WriteHeader(res.StatusCode)

	resBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		errStr := fmt.Errorf("error reading response payload: %s session ID: %s", err.Error(), sessionID)
		fmt.Println(errStr)
		_ = returnProxyError(rw, errStr.Error())
		return
	}
	defer res.Body.Close()

	reader := bytes.NewReader(resBytes)
	gzreader, err := gzip.NewReader(reader)
	if err != nil {
		errStr := fmt.Errorf("error creating gzip reader: %s session ID: %s", err.Error(), sessionID)
		fmt.Println(errStr)
		_ = returnProxyError(rw, errStr.Error())
	}

	resBytes, err = ioutil.ReadAll(gzreader)
	if err != nil {
		errStr := fmt.Errorf("error reading from gzip reader: %s session ID: %s", err.Error(), sessionID)
		fmt.Println(errStr)
		_ = returnProxyError(rw, errStr.Error())
	}

	for k, v := range res.Header {
		for i := 0; i < len(v); i++ {
			rw.Header().Add(k, v[i])
		}
	}

	_, err = rw.Write(resBytes)
	if err != nil {
		errStr := fmt.Errorf("error writing server response for client: %s session ID: %s", err.Error(), sessionID)
		fmt.Println(errStr)

		_ = returnProxyError(rw, errStr.Error())
		return
	}

	if res.StatusCode != 200 {
		rw.WriteHeader(res.StatusCode)
	}

	fmt.Printf(`
[RESPONSE ID]: %s
[STATUS]: %d
[HEADERS]:
%s
[RESPONSE BODY]:
%s

`, sessionID, res.StatusCode, headerToPrintableFormat(res.Header), string(resBytes))
}

type proxyError struct {
	ProxyErrorMessage string `json:"proxyErrorMessage"`
}

func returnProxyError(rw http.ResponseWriter, errMsg string) error {
	pe := &proxyError{ProxyErrorMessage: errMsg}
	jsonBytes, _ := json.Marshal(pe)

	rw.WriteHeader(500)

	for k, v := range rw.Header() {
		fmt.Println(k + ":" + v[0])
	}

	rw.Header().Set("Content-Type", "application/json")

	_, err := rw.Write(jsonBytes)
	return err
}

func headerToPrintableFormat(h http.Header) string {
	msg := ""
	for k, v := range h {
		if len(v) == 1 {
			msg = fmt.Sprintf("%s%s: %s\n", msg, k, v[0])
		} else {
			msg = fmt.Sprintf("%s%s: %s\n", msg, k, v)
		}
	}
	return msg
}
