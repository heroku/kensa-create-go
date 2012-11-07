package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type statusCapturingResponseWriter struct {
	status int
	http.ResponseWriter
}

func (w statusCapturingResponseWriter) WriteHeader(s int) {
	w.status = s
	w.ResponseWriter.WriteHeader(s)
}

func runLogging(logs chan string) {
	for log := range logs {
		fmt.Println(log)
	}
}

func wrapLogging(f http.HandlerFunc, logs chan string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		wres := statusCapturingResponseWriter{-1, res}
		start := time.Now()
		f(wres, req)
		method := req.Method
		path := req.URL.Path
		elapsed := float64(time.Since(start)) / 1000000.0
		logs <- fmt.Sprintf("request at=finish method=%s path=%s status=%d elapsed=%f",
			method, path, wres.status, elapsed)
	}
}

type authenticator func(string, string) bool

func testAuth(r *http.Request, auth authenticator) bool {
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 || s[0] != "Basic" {
		return false
	}
	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}
	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}
	return auth(pair[0], pair[1])
}

func denyAuth(res http.ResponseWriter) {
	res.Header().Set("WWW-Authenticate", `Basic realm="private"`)
	res.WriteHeader(401)
	res.Write([]byte("{message: \"Unauthorized\"}\n"))
}

func ensureAuth(res http.ResponseWriter, req *http.Request, auth authenticator) bool {
	if testAuth(req, auth) {
		return true
	}
	denyAuth(res)
	return false
}

func readForm(resp http.ResponseWriter, req *http.Request) bool {
	err := req.ParseForm()
	if err != nil {
		resp.WriteHeader(400)
		resp.Write([]byte("{message: \"Invalid body\"}"))
		return false
	}
	return true
}

func readJson(resp http.ResponseWriter, req *http.Request, reqD interface{}) bool {
	err := json.NewDecoder(req.Body).Decode(reqD)
	if err != nil {
		resp.WriteHeader(400)
		resp.Write([]byte("{message: \"Invalid body\"}"))
		return false
	}
	return true
}

func writeJson(resp http.ResponseWriter, respD interface{}) {
	b, err := json.Marshal(&respD)
	if err != nil {
		resp.WriteHeader(500)
		resp.Write([]byte("{message: \"Internal server error\"}"))
	} else {
		resp.Write(b)
	}
}
