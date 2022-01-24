package program

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fake-proxy/utils/stdout_utils"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"
)

const (
	defaultPort int64 = 8080
)

func Main(args []string) {
	if len(args) == 1 {
		switch {
		case args[0] == "version" || args[0] == "--version" || args[0] == "-v":
			fmt.Println(versionInfo())
			os.Exit(ErrSuccess)
			return
		case args[0] == "help" || args[0] == "--help" || args[0] == "-h":
			fmt.Println(helpMessageStyled())
			os.Exit(ErrSuccess)
		case args[0] == "-hr":
			fmt.Println(helpMessageUnstyled())
			os.Exit(ErrSuccess)
		default:
			fmt.Println(helpPrompt)
			os.Exit(ErrInput)
		}
	}

	portToUse := defaultPort
	if len(args) == 2 && args[0] == "-p" {
		inputPort, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			fmt.Printf("Invalid port number: '%v'.\n", args[1])
			os.Exit(ErrInput)
		}
		if inputPort < 0 || inputPort > 65535 {
			fmt.Printf("Port number must be between 0 - 65535.\n")
			os.Exit(ErrInput)
		}
		portToUse = inputPort
	}

	fmt.Printf("Starting proxy on port: %d\n", portToUse)
	handler := &httpHandler{}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nStopping proxy.")
		os.Exit(ErrSuccess)
	}()

	err := http.ListenAndServe(fmt.Sprintf(":%d", portToUse), handler)
	if err != nil {
		fmt.Printf("\nError occured: %s", err.Error())
		fmt.Println("\nStopping proxy.")
		os.Exit(ErrInternal)
	}
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

	stdout_utils.PrintfStyled(`
<b><yellow>[REQUEST ID]:</yellow></b> %s
<b><yellow>[URL]:</yellow></b> %s
<b><yellow>[METHOD]:</yellow></b> %s
<b><yellow>[HEADERS]:</yellow></b>
%s
<b><yellow>[REQUEST BODY]:</yellow></b>
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

	resBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		errStr := fmt.Errorf("error reading response payload: %s session ID: %s", err.Error(), sessionID)
		fmt.Println(errStr)
		_ = returnProxyError(rw, errStr.Error())
		return
	}
	defer res.Body.Close()

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

	/* Set status code last, or other header values and body will be lost. */
	if res.StatusCode != http.StatusOK {
		rw.WriteHeader(res.StatusCode)
	}

	if res.Header.Get("Content-Encoding") == "gzip" {
		reader := bytes.NewReader(resBytes)
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			errStr := fmt.Errorf("error creating gzip reader: %s session ID: %s", err.Error(), sessionID)
			fmt.Println(errStr)
			_ = returnProxyError(rw, errStr.Error())
		}
		/* Modifying resBytes for logging decompressed content AFTER we've written the response body. */
		resBytes, err = ioutil.ReadAll(gzipReader)
		if err != nil {
			errStr := fmt.Errorf("error reading from gzip reader: %s session ID: %s", err.Error(), sessionID)
			fmt.Println(errStr)
			_ = returnProxyError(rw, errStr.Error())
		}
	}

	stdout_utils.PrintfStyled(`

<b><yellow>[RESPONSE ID]:</yellow></b> %s
<b><yellow>[STATUS]:</yellow></b> %d
<b><yellow>[HEADERS]:</yellow></b>
%s
<b><yellow>[RESPONSE BODY]:</yellow></b>
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
