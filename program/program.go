package program

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/onuryurdupak/gomod/v2/slice"
	"github.com/onuryurdupak/gomod/v2/stdout"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/google/uuid"
	"github.com/yosssi/gohtml"
)

const (
	defaultPort int64 = 8080
)

type Program struct {
	out io.Writer // Where the proxy logs requests and responses.
}

func NewProgram() *Program {
	return &Program{out: os.Stdout}
}

func (p *Program) Main(args []string) (output string, code int) {
	rawMode := slice.RemoveString(&args, "--raw")

	if len(args) == 1 {
		switch args[0] {
		case "version", "--version", "-v":
			return versionInfo(), CodeSuccess
		case "help", "--help", "-h":
			return helpMessageStyled(), CodeSuccess
		case "-hr":
			return helpMessageUnstyled(), CodeSuccess
		default:
			return helpPrompt, CodeInputErr
		}
	}

	portToUse := defaultPort
	if len(args) == 2 && args[0] == "-p" {
		inputPort, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Sprintf("Invalid port number: '%v'.", args[1]), CodeInputErr
		}
		if inputPort < 0 || inputPort > 65535 {
			return "Port number must be between 0 - 65535.", CodeInputErr
		}
		portToUse = inputPort
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", portToUse),
		Handler: &httpHandler{rawMode: rawMode, out: p.out},
	}

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-signals:
			_ = server.Close()
		case <-done:
		}
	}()

	fmt.Fprintf(p.out, "Starting proxy on port: %d\n", portToUse)

	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return "\nStopping proxy.", CodeSuccess
	}
	return fmt.Sprintf("\nError occured: %s\nStopping proxy.", err.Error()), CodeInternalErr
}

type httpHandler struct {
	httpCLi http.Client
	rawMode bool
	out     io.Writer
}

func (h *httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	sessionID := uuid.New()

	redirectUrl := strings.Replace(r.URL.RequestURI(), "/", "", 1)
	parsedRedirectUrl, err := url.Parse(redirectUrl)

	if err != nil {
		errStr := fmt.Errorf("unable to parse URL: '%s' error: %w", redirectUrl, err)
		fmt.Fprintln(h.out, errStr)
		_ = returnProxyError(rw, http.StatusBadRequest, errStr.Error())
		return
	}

	reqBytes, err := io.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Errorf("error reading request body: %w", err)
		fmt.Fprintln(h.out, errStr)
		_ = returnProxyError(rw, http.StatusBadRequest, errStr.Error())
		return
	}
	defer r.Body.Close()

	printStyled(h.out, `
<b><yellow>[REQUEST ID]:</yellow></b> %s
<b><yellow>[URL]:</yellow></b> %s
<b><yellow>[METHOD]:</yellow></b> %s
<b><yellow>[HEADERS]</yellow></b>
%s
<b><yellow>[REQUEST BODY]</yellow></b>
%s`, sessionID, redirectUrl, r.Method, headerToPrintableFormat(r.Header), bodyToPrintableFormat(r.Header, reqBytes, h.rawMode))

	buffer := bytes.NewBuffer(reqBytes)
	nopCloser := io.NopCloser(buffer)

	httpReq := &http.Request{
		Method: r.Method,
		URL:    parsedRedirectUrl,
		Header: r.Header,
		Body:   nopCloser,
		/* Go's client ignores the Content-Length header and sends bodies chunked unless this is set. */
		ContentLength: int64(len(reqBytes)),
	}

	res, err := h.httpCLi.Do(httpReq)
	if err != nil {
		errStr := fmt.Errorf("error executing http request: %s session ID: %s", err.Error(), sessionID)
		fmt.Fprintln(h.out, errStr)
		_ = returnProxyError(rw, http.StatusBadGateway, errStr.Error())
		return
	}
	defer res.Body.Close()

	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		errStr := fmt.Errorf("error reading response payload: %s session ID: %s", err.Error(), sessionID)
		fmt.Fprintln(h.out, errStr)
		_ = returnProxyError(rw, http.StatusBadGateway, errStr.Error())
		return
	}

	for k, v := range res.Header {
		for i := 0; i < len(v); i++ {
			rw.Header().Add(k, v[i])
		}
	}

	rw.WriteHeader(res.StatusCode)

	_, err = rw.Write(resBytes)
	if err != nil {
		/* Status and headers are already sent, so the client can not be told about this. */
		errStr := fmt.Errorf("error writing server response for client: %s session ID: %s", err.Error(), sessionID)
		fmt.Fprintln(h.out, errStr)
		return
	}

	/* The client already has the response, so from here on errors only affect logging and are not sent to the client. */
	var printableResBody string
	if res.Header.Get("Content-Encoding") == "gzip" {
		decompressed, err := gunzip(resBytes)
		if err != nil {
			printableResBody = fmt.Sprintf("(could not decompress gzip body for logging: %s)", err.Error())
		} else {
			printableResBody = bodyToPrintableFormat(res.Header, decompressed, h.rawMode)
		}
	} else {
		printableResBody = bodyToPrintableFormat(res.Header, resBytes, h.rawMode)
	}

	printStyled(h.out, `

<b><yellow>[RESPONSE ID]:</yellow></b> %s
<b><yellow>[STATUS]:</yellow></b> %d
<b><yellow>[HEADERS]</yellow></b> 
%s
<b><yellow>[RESPONSE BODY]</yellow></b>
%s
`, sessionID, res.StatusCode, headerToPrintableFormat(res.Header), printableResBody)
}

func gunzip(data []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	return io.ReadAll(gzipReader)
}

type proxyError struct {
	ProxyErrorMessage string `json:"proxyErrorMessage"`
}

func returnProxyError(rw http.ResponseWriter, statusCode int, errMsg string) error {
	pe := &proxyError{ProxyErrorMessage: errMsg}
	jsonBytes, _ := json.Marshal(pe)

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)

	_, err := rw.Write(jsonBytes)
	return err
}

// printStyled applies style tags in format only, then fills in args.
// Args such as headers and bodies are printed as is, so tags like <b> inside them are not interpreted.
func printStyled(w io.Writer, format string, args ...any) {
	styledFormat, _ := stdout.ProcessStyle(format)
	fmt.Fprintf(w, styledFormat, args...)
}

// style applies style tags to a string that contains no user data.
func style(s string) string {
	styled, _ := stdout.ProcessStyle(s)
	return styled
}

func headerToPrintableFormat(h http.Header) string {
	lines := make([]string, 0, len(h))
	for _, k := range slices.Sorted(maps.Keys(h)) {
		lines = append(lines, style("<green>"+k+":</green>")+" "+strings.Join(h[k], ", "))
	}
	return strings.Join(lines, "\n")
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
		err := json.Indent(dst, body, "", "  ")
		if err != nil {
			return string(body)
		}
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
