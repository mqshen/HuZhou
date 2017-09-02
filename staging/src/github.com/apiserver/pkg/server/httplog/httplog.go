package httplog

import (
	"net/http"
	"time"
	"github.com/golang/glog"
	"fmt"
	"bufio"
	"net"
	"runtime"
)


type logger interface {
	Addf(format string, data ...interface{})
}

// Simple logger that logs immediately when Addf is called
type passthroughLogger struct{}

// Addf logs info immediately.
func (passthroughLogger) Addf(format string, data ...interface{}) {
	glog.V(2).Info(fmt.Sprintf(format, data...))
}

// StacktracePred returns true if a stacktrace should be logged for this status.
type StacktracePred func(httpStatus int) (logStacktrace bool)

type respLogger struct {
	hijacked       bool
	statusRecorded bool
	status         int
	statusStack    string
	addedInfo      string
	startTime      time.Time

	captureErrorOutput bool

	req *http.Request
	w   http.ResponseWriter

	logStacktracePred StacktracePred
}


// StacktraceWhen sets the stacktrace logging predicate, which decides when to log a stacktrace.
// There's a default, so you don't need to call this unless you don't like the default.
func (rl *respLogger) StacktraceWhen(pred StacktracePred) *respLogger {
	rl.logStacktracePred = pred
	return rl
}

// DefaultStacktracePred is the default implementation of StacktracePred.
func DefaultStacktracePred(status int) bool {
	return (status < http.StatusOK || status >= http.StatusInternalServerError) && status != http.StatusSwitchingProtocols
}

func NewLogged(req *http.Request, w *http.ResponseWriter) *respLogger {
	if _, ok := (*w).(*respLogger); ok {
		// Don't double-wrap!
		panic("multiple NewLogged calls!")
	}
	rl := &respLogger{
		startTime:         time.Now(),
		req:               req,
		w:                 *w,
		logStacktracePred: DefaultStacktracePred,
	}
	*w = rl // hijack caller's writer!
	return rl
}


// Addf adds additional data to be logged with this request.
func (rl *respLogger) Addf(format string, data ...interface{}) {
	rl.addedInfo += "\n" + fmt.Sprintf(format, data...)
}

// Log is intended to be called once at the end of your request handler, via defer
func (rl *respLogger) Log() {
	latency := time.Since(rl.startTime)
	if glog.V(2) {
		if !rl.hijacked {
			glog.InfoDepth(1, fmt.Sprintf("%s %s: (%v) %v%v%v [%s %s]", rl.req.Method, rl.req.RequestURI, latency, rl.status, rl.statusStack, rl.addedInfo, rl.req.Header["User-Agent"], rl.req.RemoteAddr))
		} else {
			glog.InfoDepth(1, fmt.Sprintf("%s %s: (%v) hijacked [%s %s]", rl.req.Method, rl.req.RequestURI, latency, rl.req.Header["User-Agent"], rl.req.RemoteAddr))
		}
	}
}

// Header implements http.ResponseWriter.
func (rl *respLogger) Header() http.Header {
	return rl.w.Header()
}

// Write implements http.ResponseWriter.
func (rl *respLogger) Write(b []byte) (int, error) {
	if !rl.statusRecorded {
		rl.recordStatus(http.StatusOK) // Default if WriteHeader hasn't been called
	}
	if rl.captureErrorOutput {
		rl.Addf("logging error output: %q\n", string(b))
	}
	return rl.w.Write(b)
}

// Flush implements http.Flusher even if the underlying http.Writer doesn't implement it.
// Flush is used for streaming purposes and allows to flush buffered data to the client.
func (rl *respLogger) Flush() {
	if flusher, ok := rl.w.(http.Flusher); ok {
		flusher.Flush()
	} else if glog.V(2) {
		glog.InfoDepth(1, fmt.Sprintf("Unable to convert %+v into http.Flusher", rl.w))
	}
}

// WriteHeader implements http.ResponseWriter.
func (rl *respLogger) WriteHeader(status int) {
	rl.recordStatus(status)
	rl.w.WriteHeader(status)
}

// Hijack implements http.Hijacker.
func (rl *respLogger) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rl.hijacked = true
	return rl.w.(http.Hijacker).Hijack()
}

// CloseNotify implements http.CloseNotifier
func (rl *respLogger) CloseNotify() <-chan bool {
	return rl.w.(http.CloseNotifier).CloseNotify()
}

func (rl *respLogger) recordStatus(status int) {
	rl.status = status
	rl.statusRecorded = true
	if rl.logStacktracePred(status) {
		// Only log stacks for errors
		stack := make([]byte, 50*1024)
		stack = stack[:runtime.Stack(stack, false)]
		rl.statusStack = "\n" + string(stack)
		rl.captureErrorOutput = true
	} else {
		rl.statusStack = ""
	}
}

// LogOf returns the logger hiding in w. If there is not an existing logger
// then a passthroughLogger will be created which will log to stdout immediately
// when Addf is called.
func LogOf(req *http.Request, w http.ResponseWriter) logger {
	if _, exists := w.(*respLogger); !exists {
		pl := &passthroughLogger{}
		return pl
	}
	if rl, ok := w.(*respLogger); ok {
		return rl
	}
	panic("Unable to find or create the logger!")
}