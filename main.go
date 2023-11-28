package chizerolog

import (
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

type LoggerOpts struct {
	Message                 string
	AccessLogTypeName       string
	PrintLogType            bool
	PrintStackTraceToStderr bool
	CustomFields            map[string]func(ww middleware.WrapResponseWriter, r *http.Request) interface{}
}

func LoggerMiddleware(logger *zerolog.Logger, opts *LoggerOpts) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = &LoggerOpts{
			Message:                 "incoming_request",
			AccessLogTypeName:       "access",
			PrintLogType:            true,
			PrintStackTraceToStderr: true,
		}
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			zlog := logger.With().Logger()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				// Recover and record stack traces in case of a panic
				if rec := recover(); rec != nil {
					if opts.PrintStackTraceToStderr {
						debug.PrintStack()
					} else {
						le := zlog.Error()
						if opts != nil && opts.PrintLogType && opts.AccessLogTypeName != "" {
							le = le.Str("type", "error")
						}
						le.Timestamp().
							Interface("recover_info", rec).
							Bytes("debug_stack", debug.Stack()).
							Msg("log system error")
					}
					http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}

				// log end request
				le := zlog.Info()
				if opts != nil && opts.PrintLogType && opts.AccessLogTypeName != "" {
					le = le.Str("type", opts.AccessLogTypeName)
				}
				le.Timestamp().
					Fields(map[string]interface{}{
						"remote_ip":  getRemoteAddr(r),
						"url":        r.URL.Path,
						"proto":      r.Proto,
						"method":     r.Method,
						"user_agent": r.Header.Get("User-Agent"),
						"status":     ww.Status(),
						"latency":    t2.Sub(t1).Truncate(1000 * time.Nanosecond).String(),
						"bytes_in":   r.Header.Get("Content-Length"),
						"bytes_out":  ww.BytesWritten(),
					})
				if len(opts.CustomFields) > 0 {
					for fieldName, valueFunc := range opts.CustomFields {
						le = le.Fields(map[string]interface{}{
							fieldName: valueFunc(ww, r),
						})
					}
				}
				le.Msg(opts.Message)
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

func getRemoteAddr(r *http.Request) string {
	remoteAddr := r.RemoteAddr
	if strings.Contains(remoteAddr, ":") {
		var err error
		remoteAddr, _, err = net.SplitHostPort(remoteAddr)
		if err != nil {
			log.Error().Err(err).Msg("could not parse remote address")
		}
	}
	return remoteAddr
}
