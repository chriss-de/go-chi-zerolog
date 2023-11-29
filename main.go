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
	Fields                  map[string]func(ww middleware.WrapResponseWriter, r *http.Request) interface{}
}

func DefaultLoggerOpts() *LoggerOpts {
	return &LoggerOpts{
		Message:                 "incoming_request",
		AccessLogTypeName:       "access",
		PrintLogType:            true,
		PrintStackTraceToStderr: true,
		Fields: map[string]func(ww middleware.WrapResponseWriter, r *http.Request) interface{}{
			"remote_ip":  remoteAddr,
			"url":        url,
			"proto":      proto,
			"method":     method,
			"user_agent": userAgent,
			"status":     status,
			//"latency":    latency,
			"bytes_in":  bytesIn,
			"bytes_out": bytesOut,
		},
	}
}

func LoggerMiddleware(logger *zerolog.Logger, opts *LoggerOpts) func(next http.Handler) http.Handler {
	if opts == nil {
		opts = DefaultLoggerOpts()
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
				le := zlog.Info().Timestamp()
				if opts != nil && opts.PrintLogType && opts.AccessLogTypeName != "" {
					le = le.Str("type", opts.AccessLogTypeName)
				}

				// TODO
				le.Str("latency", t2.Sub(t1).Truncate(1000*time.Nanosecond).String())

				if len(opts.Fields) > 0 {
					for fieldName, valueFunc := range opts.Fields {
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
