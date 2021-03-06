// Copyright 2017 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/liudf0716/xfrps/assets"
	"github.com/liudf0716/xfrps/models/config"

	"github.com/julienschmidt/httprouter"
)

var (
	httpServerReadTimeout  = 10 * time.Second
	httpServerWriteTimeout = 10 * time.Second
)

func RunDashboardServer(addr string, port int64) (err error) {
	// url router
	router := httprouter.New()

	// api, see dashboard_api.go
	router.GET("/api/serverinfo", httprouterBasicAuth(apiServerInfo))
	router.GET("/api/proxy/tcp", httprouterBasicAuth(apiProxyTcp))
	router.GET("/api/proxy/udp", httprouterBasicAuth(apiProxyUdp))
	router.GET("/api/proxy/ftp", httprouterBasicAuth(apiProxyFtp))
	router.GET("/api/proxy/http", httprouterBasicAuth(apiProxyHttp))
	router.GET("/api/proxy/https", httprouterBasicAuth(apiProxyHttps))
	router.GET("/api/proxy/tcp/:pageNo", httprouterBasicAuth(apiProxyTcp))
	router.GET("/api/proxy/udp/:pageNo", httprouterBasicAuth(apiProxyUdp))
	router.GET("/api/proxy/ftp/:pageNo", httprouterBasicAuth(apiProxyFtp))
	router.GET("/api/proxy/http/:pageNo", httprouterBasicAuth(apiProxyHttp))
	router.GET("/api/proxy/https/:pageNo", httprouterBasicAuth(apiProxyHttps))
	router.GET("/api/proxy/traffic/:name", httprouterBasicAuth(apiProxyTraffic))
	router.GET("/api/client/online", httprouterBasicAuth(apiClientOnline))
	router.GET("/api/client/online/:pageNo", httprouterBasicAuth(apiClientOnline))
	router.GET("/api/client/offline", httprouterBasicAuth(apiClientOffline))
	router.GET("/api/client/offline/:pageNo", httprouterBasicAuth(apiClientOffline))
	router.GET("/api/port/getfree/:proto", httprouterNoAuth(apiGetFreePort))
	router.GET("/api/port/tcp/isfree/:port", httprouterNoAuth(apiIsTcpPortFree))
	router.GET("/api/port/tcp/getport/:runid", httprouterNoAuth(apiGetPort))       // according runid, getting tcp port
	router.GET("/api/port/tcp/getftpport/:runid", httprouterNoAuth(apiGetFtpPort)) // according runid, getting its ftp control port

	// view
	router.Handler("GET", "/favicon.ico", http.FileServer(assets.FileSystem))
	router.Handler("GET", "/static/*filepath", MakeGzipHandler(basicAuthWraper(http.StripPrefix("/static/", http.FileServer(assets.FileSystem)))))
	router.HandlerFunc("GET", "/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusMovedPermanently)
	}))

	address := fmt.Sprintf("%s:%d", addr, port)
	server := &http.Server{
		Addr:         address,
		Handler:      router,
		ReadTimeout:  httpServerReadTimeout,
		WriteTimeout: httpServerWriteTimeout,
	}
	if address == "" {
		address = ":http"
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	go server.Serve(ln)
	return
}

func use(h http.HandlerFunc, middleware ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middleware {
		h = m(h)
	}
	return h
}

type AuthWraper struct {
	h      http.Handler
	user   string
	passwd string
}

func (aw *AuthWraper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, passwd, hasAuth := r.BasicAuth()
	if (aw.user == "" && aw.passwd == "") || (hasAuth && user == aw.user && passwd == aw.passwd) {
		aw.h.ServeHTTP(w, r)
	} else {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
}

func basicAuthWraper(h http.Handler) http.Handler {
	return &AuthWraper{
		h:      h,
		user:   config.ServerCommonCfg.DashboardUser,
		passwd: config.ServerCommonCfg.DashboardPwd,
	}
}

func basicAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, passwd, hasAuth := r.BasicAuth()
		if (config.ServerCommonCfg.DashboardUser == "" && config.ServerCommonCfg.DashboardPwd == "") ||
			(hasAuth && user == config.ServerCommonCfg.DashboardUser && passwd == config.ServerCommonCfg.DashboardPwd) {
			h.ServeHTTP(w, r)
		} else {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}
}

func httprouterBasicAuth(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		user, passwd, hasAuth := r.BasicAuth()
		if (config.ServerCommonCfg.DashboardUser == "" && config.ServerCommonCfg.DashboardPwd == "") ||
			(hasAuth && user == config.ServerCommonCfg.DashboardUser && passwd == config.ServerCommonCfg.DashboardPwd) {
			h(w, r, ps)
		} else {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}
}

func httprouterNoAuth(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h(w, r, ps)
	}
}

type GzipWraper struct {
	h http.Handler
}

func (gw *GzipWraper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		gw.h.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
	gw.h.ServeHTTP(gzr, r)
}

func MakeGzipHandler(h http.Handler) http.Handler {
	return &GzipWraper{
		h: h,
	}
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}
