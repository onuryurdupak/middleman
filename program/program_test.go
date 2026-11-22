package program

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
)

func newTestHandler() (*httpHandler, *bytes.Buffer) {
	log := &bytes.Buffer{}
	return &httpHandler{out: log}, log
}

func gzipBytes(t *testing.T, data string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := gzip.NewWriter(&b)
	if _, err := zw.Write([]byte(data)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func assertProxyError(t *testing.T, rec *httptest.ResponseRecorder, wantCode int, wantPrefix string) {
	t.Helper()
	if rec.Code != wantCode {
		t.Errorf("code = %d, want %d", rec.Code, wantCode)
	}
	// Result() has the headers as sent with the status line; the recorder's live map would also show headers set too late.
	if got := rec.Result().Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
	var pe proxyError
	if err := json.Unmarshal(rec.Body.Bytes(), &pe); err != nil {
		t.Fatalf("body %q is not a proxy error: %v", rec.Body.String(), err)
	}
	if !strings.HasPrefix(pe.ProxyErrorMessage, wantPrefix) {
		t.Errorf("proxyErrorMessage = %q, want prefix %q", pe.ProxyErrorMessage, wantPrefix)
	}
}

// failingResponseWriter accepts headers but fails every body write, like a client that disconnected.
type failingResponseWriter struct {
	header      http.Header
	statusCodes []int
	writes      int
}

func (w *failingResponseWriter) Header() http.Header { return w.header }

func (w *failingResponseWriter) WriteHeader(code int) { w.statusCodes = append(w.statusCodes, code) }

func (w *failingResponseWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errors.New("client went away")
}

func TestMainSingleArgument(t *testing.T) {
	tests := []struct {
		arg        string
		wantOutput string
		wantCode   int
	}{
		{"version", versionInfo(), CodeSuccess},
		{"--version", versionInfo(), CodeSuccess},
		{"-v", versionInfo(), CodeSuccess},
		{"help", helpMessageStyled(), CodeSuccess},
		{"--help", helpMessageStyled(), CodeSuccess},
		{"-h", helpMessageStyled(), CodeSuccess},
		{"-hr", helpMessageUnstyled(), CodeSuccess},
		{"-x", helpPrompt, CodeInputErr},
		{"-p", helpPrompt, CodeInputErr},
		{"", helpPrompt, CodeInputErr},
	}

	for _, tt := range tests {
		t.Run(strconv.Quote(tt.arg), func(t *testing.T) {
			output, code := NewProgram().Main([]string{tt.arg})
			if code != tt.wantCode {
				t.Errorf("code = %d, want %d", code, tt.wantCode)
			}
			if output != tt.wantOutput {
				t.Errorf("output = %q, want %q", output, tt.wantOutput)
			}
		})
	}
}

func TestMainRawFlagIsRemovedBeforeParsing(t *testing.T) {
	tests := [][]string{
		{"--raw", "-v"},
		{"-v", "--raw"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			output, code := NewProgram().Main(args)
			if code != CodeSuccess || output != versionInfo() {
				t.Errorf("got (%q, %d), want version info and %d", output, code, CodeSuccess)
			}
		})
	}
}

func TestMainInvalidPort(t *testing.T) {
	const outOfRange = "Port number must be between 0 - 65535."

	tests := []struct {
		args       []string
		wantOutput string
	}{
		{[]string{"-p", "abc"}, "Invalid port number: 'abc'."},
		{[]string{"-p", "50.5"}, "Invalid port number: '50.5'."},
		{[]string{"-p", "99999999999999999999"}, "Invalid port number: '99999999999999999999'."},
		{[]string{"-p", "-1"}, outOfRange},
		{[]string{"-p", "65536"}, outOfRange},
		{[]string{"--raw", "-p", "abc"}, "Invalid port number: 'abc'."},
		{[]string{"-p", "abc", "--raw"}, "Invalid port number: 'abc'."},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			p := &Program{out: io.Discard}
			output, code := p.Main(tt.args)
			if code != CodeInputErr {
				t.Errorf("code = %d, want %d", code, CodeInputErr)
			}
			if output != tt.wantOutput {
				t.Errorf("output = %q, want %q", output, tt.wantOutput)
			}
		})
	}
}

func TestHelpMessageUnstyledHasNoStyleTags(t *testing.T) {
	msg := helpMessageUnstyled()
	for _, tag := range []string{"<b>", "</b>", "<u>", "</u>", "<yellow>", "</yellow>"} {
		if strings.Contains(msg, tag) {
			t.Errorf("unstyled help contains %q", tag)
		}
	}
	if !strings.Contains(msg, "PARAMETERS:") {
		t.Errorf("unstyled help is missing %q", "PARAMETERS:")
	}
}

func TestBodyToPrintableFormat(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		rawMode     bool
		want        string
	}{
		{"json is indented", "application/json", `{"a":1}`, false, "{\n  \"a\": 1\n}"},
		{"json with charset is indented", "application/json; charset=utf-8", `[1,2]`, false, "[\n  1,\n  2\n]"},
		{"invalid json is shown raw", "application/json", `{"a": 1, broken`, false, `{"a": 1, broken`},
		{"empty json body stays empty", "application/json", "", false, ""},
		{"raw mode skips formatting", "application/json", `{"a":1}`, true, `{"a":1}`},
		{"no content type is shown raw", "", `{"a":1}`, false, `{"a":1}`},
		{"other content type is shown raw", "text/plain", "hello\n  world", false, "hello\n  world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			if tt.contentType != "" {
				h.Set("Content-Type", tt.contentType)
			}
			if got := bodyToPrintableFormat(h, []byte(tt.body), tt.rawMode); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBodyToPrintableFormatXMLAndHTML(t *testing.T) {
	tests := []struct {
		contentType string
		body        string
	}{
		{"application/xml", "<a><b>1</b></a>"},
		{"text/html", "<html><body><p>hi</p></body></html>"},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			h := http.Header{"Content-Type": {tt.contentType}}
			got := bodyToPrintableFormat(h, []byte(tt.body), false)
			if got == tt.body {
				t.Errorf("body was not formatted: %q", got)
			}
			if strings.Join(strings.Fields(got), "") != strings.Join(strings.Fields(tt.body), "") {
				t.Errorf("formatting changed more than whitespace: %q", got)
			}
			if raw := bodyToPrintableFormat(h, []byte(tt.body), true); raw != tt.body {
				t.Errorf("raw mode = %q, want %q", raw, tt.body)
			}
		})
	}
}

func TestHeaderToPrintableFormat(t *testing.T) {
	h := http.Header{}
	h.Set("X-Trace", "abc")
	h.Set("Content-Type", "application/json")
	h.Add("Accept", "text/html")
	h.Add("Accept", "application/json")

	want := style("<green>Accept:</green>") + " text/html, application/json\n" +
		style("<green>Content-Type:</green>") + " application/json\n" +
		style("<green>X-Trace:</green>") + " abc"

	if got := headerToPrintableFormat(h); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := headerToPrintableFormat(http.Header{}); got != "" {
		t.Errorf("empty header = %q, want empty", got)
	}
}

func TestServeHTTPForwardsRequestAndResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost {
			t.Errorf("upstream method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/items" || r.URL.RawQuery != "q=1&r=2" {
			t.Errorf("upstream URL = %q, want /items?q=1&r=2", r.URL.RequestURI())
		}
		if got := r.Header.Get("X-Client"); got != "yes" {
			t.Errorf("upstream X-Client = %q, want yes", got)
		}
		if string(body) != `{"name":"x"}` {
			t.Errorf("upstream body = %q", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("X-Upstream", "a")
		w.Header().Add("X-Upstream", "b")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":7}`))
	}))
	defer upstream.Close()

	handler, log := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/"+upstream.URL+"/items?q=1&r=2", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client", "yes")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("code = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Result().Header.Values("X-Upstream"); strings.Join(got, ",") != "a,b" {
		t.Errorf("X-Upstream = %v, want [a b]", got)
	}
	if rec.Body.String() != `{"id":7}` {
		t.Errorf("client body = %q, want %q", rec.Body.String(), `{"id":7}`)
	}

	for _, want := range []string{
		upstream.URL + "/items?q=1&r=2",
		"POST",
		"{\n  \"name\": \"x\"\n}",
		"201",
		"{\n  \"id\": 7\n}",
		" a, b",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log is missing %q:\n%s", want, log.String())
		}
	}
}

func TestServeHTTPLogsStyleTagsLiterally(t *testing.T) {
	const body = "<b>bold?</b> <yellow>unclosed"
	const headerValue = "<red>header</red>"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Note", headerValue)
		_, _ = w.Write([]byte(body))
	}))
	defer upstream.Close()

	handler, log := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/"+upstream.URL, strings.NewReader("<u>request</u>"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Body.String() != body {
		t.Errorf("client body = %q, want %q", rec.Body.String(), body)
	}
	for _, want := range []string{body, headerValue, "<u>request</u>"} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log is missing %q:\n%s", want, log.String())
		}
	}
}

func TestServeHTTPGzip(t *testing.T) {
	compressed := gzipBytes(t, `{"ok":true}`)

	tests := []struct {
		name     string
		upstream []byte
		wantLog  string
	}{
		{"valid gzip is logged decompressed", compressed, "{\n  \"ok\": true\n}"},
		{"invalid gzip is reported in the log only", []byte("not gzip"), "(could not decompress gzip body for logging: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Content-Encoding", "gzip")
				_, _ = w.Write(tt.upstream)
			}))
			defer upstream.Close()

			handler, log := newTestHandler()
			req := httptest.NewRequest(http.MethodGet, "/"+upstream.URL, nil)
			// Sent by the client, so Go's transport passes the body through instead of decompressing it.
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("code = %d, want %d", rec.Code, http.StatusOK)
			}
			if !bytes.Equal(rec.Body.Bytes(), tt.upstream) {
				t.Errorf("client body = %q, want the upstream body %q unchanged", rec.Body.Bytes(), tt.upstream)
			}
			if got := rec.Result().Header.Get("Content-Encoding"); got != "gzip" {
				t.Errorf("Content-Encoding = %q, want gzip", got)
			}
			if !strings.Contains(log.String(), tt.wantLog) {
				t.Errorf("log is missing %q:\n%s", tt.wantLog, log.String())
			}
		})
	}
}

func TestServeHTTPInvalidURL(t *testing.T) {
	handler, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/http://[::1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertProxyError(t, rec, http.StatusBadRequest, "unable to parse URL: 'http://[::1'")
}

func TestServeHTTPRequestBodyReadError(t *testing.T) {
	handler, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/http://127.0.0.1:1/x", nil)
	req.Body = io.NopCloser(iotest.ErrReader(errors.New("boom")))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertProxyError(t, rec, http.StatusBadRequest, "error reading request body: boom")
}

func TestServeHTTPUpstreamUnreachable(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	upstreamURL := upstream.URL
	upstream.Close()

	handler, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/"+upstreamURL, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertProxyError(t, rec, http.StatusBadGateway, "error executing http request: ")
}

func TestServeHTTPUpstreamResponseCutShort(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("short"))
	}))
	defer upstream.Close()

	handler, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/"+upstream.URL, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertProxyError(t, rec, http.StatusBadGateway, "error reading response payload: ")
}

func TestServeHTTPClientWriteFails(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("payload"))
	}))
	defer upstream.Close()

	handler, log := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/"+upstream.URL, nil)
	rw := &failingResponseWriter{header: http.Header{}}

	handler.ServeHTTP(rw, req)

	if len(rw.statusCodes) != 1 || rw.statusCodes[0] != http.StatusAccepted {
		t.Errorf("status codes written = %v, want only %d", rw.statusCodes, http.StatusAccepted)
	}
	if rw.writes != 1 {
		t.Errorf("body writes = %d, want 1 (no proxy error after the failed write)", rw.writes)
	}
	if want := "error writing server response for client: client went away"; !strings.Contains(log.String(), want) {
		t.Errorf("log is missing %q:\n%s", want, log.String())
	}
}
