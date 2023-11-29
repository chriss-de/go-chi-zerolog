package chizerolog

import (
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
	"strings"
)

func url(ww middleware.WrapResponseWriter, r *http.Request) interface{} {
	return r.URL.Path
}

func proto(ww middleware.WrapResponseWriter, r *http.Request) interface{} {
	return r.Proto
}

func method(ww middleware.WrapResponseWriter, r *http.Request) interface{} {
	return r.Method
}
func userAgent(ww middleware.WrapResponseWriter, r *http.Request) interface{} {
	return r.Header.Get("User-Agent")
}

func status(ww middleware.WrapResponseWriter, r *http.Request) interface{} {
	return ww.Status()
}

//func latency(ww middleware.WrapResponseWriter, r *http.Request) interface{} {
//	return t2.Sub(t1).Truncate(1000 * time.Nanosecond).String()
//}

func bytesIn(ww middleware.WrapResponseWriter, r *http.Request) interface{} {
	return r.Header.Get("Content-Length")
}

func bytesOut(ww middleware.WrapResponseWriter, r *http.Request) interface{} {
	return ww.BytesWritten()
}

func remoteAddr(ww middleware.WrapResponseWriter, r *http.Request) interface{} {
	rAddr := r.RemoteAddr
	if strings.Contains(rAddr, ":") {
		var err error
		rAddr, _, err = net.SplitHostPort(rAddr)
		if err != nil {
			log.Error().Err(err).Msg("could not parse remote address")
		}
	}
	return rAddr
}
