package program

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/onuryurdupak/gomod/slice"
	"github.com/onuryurdupak/gomod/stdout"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/google/uuid"
	"github.com/yosssi/gohtml"
)

const (
	defaultPort int64 = 8080
)

var rawMode bool

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
		case args[0] == "--raw":
			rawMode = slice.RemoveString(&args, "--raw")
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

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionID := uuid.New()

	destionationURL := r.URL.String()

	parsedDestinationUrl, err := url.Parse(destionationURL)
	if err != nil {
		errStr := fmt.Errorf("unable to parse URL: '%s' error: %w", destionationURL, err)
		fmt.Println(errStr)
		_ = returnProxyError(w, errStr.Error())
		return
	}

	reqBytes, err := io.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Errorf("error reading request body: %w", err)
		fmt.Println(errStr)
		_ = returnProxyError(w, errStr.Error())
		return
	}
	defer r.Body.Close()

	stdout.PrintfStyled(`
<b><yellow>[REQUEST ID]:</yellow></b> %s
<b><yellow>[URL]:</yellow></b> %s
<b><yellow>[METHOD]:</yellow></b> %s
<b><yellow>[HEADERS]</yellow></b>
%s
<b><yellow>[REQUEST BODY]</yellow></b>
%s`, sessionID, destionationURL, r.Method, headerToPrintableFormat(r.Header), bodyToPrintableFormat(r.Header, reqBytes, rawMode))

	buffer := bytes.NewBuffer(reqBytes)
	nopCloser := io.NopCloser(buffer)

	httpReq := &http.Request{
		Method: r.Method,
		URL:    parsedDestinationUrl,
		Header: r.Header,
		Body:   nopCloser,
	}

	if httpReq.Method == http.MethodConnect {
		w.WriteHeader(http.StatusOK)
		return
	}

	res, err := h.httpCLi.Do(httpReq)
	if err != nil {
		errStr := fmt.Errorf("error executing http request: %s session ID: %s", err.Error(), sessionID)
		fmt.Println(errStr)
		_ = returnProxyError(w, errStr.Error())
		return
	}
	defer res.Body.Close()

	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		errStr := fmt.Errorf("error reading response payload: %s session ID: %s", err.Error(), sessionID)
		fmt.Println(errStr)
		_ = returnProxyError(w, errStr.Error())
		return
	}
	defer res.Body.Close()

	for k, v := range res.Header {
		for i := 0; i < len(v); i++ {
			w.Header().Add(k, v[i])
		}
	}

	w.WriteHeader(res.StatusCode)

	_, err = w.Write(resBytes)
	if err != nil {
		errStr := fmt.Errorf("error writing server response for client: %s session ID: %s", err.Error(), sessionID)
		fmt.Println(errStr)
		_ = returnProxyError(w, errStr.Error())
		return
	}

	if res.Header.Get("Content-Encoding") == "gzip" {
		reader := bytes.NewReader(resBytes)
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			errStr := fmt.Errorf("error creating gzip reader: %s session ID: %s", err.Error(), sessionID)
			fmt.Println(errStr)
			_ = returnProxyError(w, errStr.Error())
		}
		/* Modifying resBytes for logging decompressed content AFTER we've written the response body. */
		resBytes, err = io.ReadAll(gzipReader)
		if err != nil {
			errStr := fmt.Errorf("error reading from gzip reader: %s session ID: %s", err.Error(), sessionID)
			fmt.Println(errStr)
			_ = returnProxyError(w, errStr.Error())
		}
	}

	stdout.PrintfStyled(`

<b><yellow>[RESPONSE ID]:</yellow></b> %s
<b><yellow>[STATUS]:</yellow></b> %d
<b><yellow>[HEADERS]</yellow></b> 
%s
<b><yellow>[RESPONSE BODY]</yellow></b>
%s
`, sessionID, res.StatusCode, headerToPrintableFormat(res.Header), bodyToPrintableFormat(res.Header, resBytes, rawMode))
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
	var sb strings.Builder
	i := 0
	length := len(h)
	for k, v := range h {
		if len(v) == 1 {
			sb.WriteString(fmt.Sprintf("<green>%s:</green> %s", k, v[0]))
		} else {
			sb.WriteString(fmt.Sprintf("<green>%s:</green> %s", k, v))
		}

		if i != length-1 {
			sb.WriteString("\n")
		}

		i++
	}
	return sb.String()
}

func bodyToPrintableFormat(h http.Header, body []byte, rawMode bool) string {
	if rawMode {
		return string(body)
	}

	contentTypeHeader := h["Content-Type"]
	if len(contentTypeHeader) < 1 {
		return string(body)
	}
	contentTypeValue := contentTypeHeader[0]
	var marshalled []byte

	if strings.Contains(contentTypeValue, "json") {
		dst := &bytes.Buffer{}
		json.Indent(dst, body, "", "  ")
		marshalled = dst.Bytes()
	} else if strings.Contains(contentTypeValue, "xml") {
		marshalled = []byte(xmlfmt.FormatXML(string(body), "", "  ", true))
	} else if strings.Contains(contentTypeValue, "html") {
		marshalled = []byte(gohtml.Format(string(body)))
	} else {
		return string(body)
	}
	return string(marshalled)
}
