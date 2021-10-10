package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	redirectUrl := strings.Replace(r.URL.RequestURI(), "/", "", 1)
	parsedRedirectUrl, err := url.Parse(redirectUrl)

	if err != nil {
		errStr := fmt.Errorf("unable to parse URL: '%s' error: %w", redirectUrl, err)
		fmt.Println(errStr)

		_ = returnProxyError(rw, errStr.Error())
		return
	}

	if err != nil {
		errStr := fmt.Errorf("error reading request body: %w", err)
		fmt.Println(errStr)

		_ = returnProxyError(rw, errStr.Error())
		return
	}

	httpReq := &http.Request{
		Method: r.Method,
		URL:    parsedRedirectUrl,
		Header: r.Header,
		Body:   r.Body,
	}

	res, err := h.httpCLi.Do(httpReq)

	if err != nil {
		errStr := fmt.Errorf("error executing http request: %w", err)
		fmt.Println(errStr)

		_ = returnProxyError(rw, errStr.Error())
		return
	}

	rw.WriteHeader(res.StatusCode)

	resBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		errStr := fmt.Errorf("error reading response payload: %w", err)
		fmt.Println(errStr)

		_ = returnProxyError(rw, errStr.Error())
		return
	}

	_, err = rw.Write(resBytes)

	if err != nil {
		errStr := fmt.Errorf("error writing server response for client: %w", err)
		fmt.Println(errStr)

		_ = returnProxyError(rw, errStr.Error())
		return
	}

	fmt.Printf("Redirected request to: '%s' and received status code: '%d'", redirectUrl, res.StatusCode)
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
