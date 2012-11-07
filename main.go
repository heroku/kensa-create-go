package main

import (
	"code.google.com/p/gorilla/mux"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func config(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing " + k)
	}
	return v
}

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

type Authenticator func(string, string) bool

func testAuth(r *http.Request, auth Authenticator) bool {
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
	res.Write([]byte("401 Unauthorized\n"))
}

func ensureAuth(res http.ResponseWriter, req *http.Request, auth Authenticator) bool {
	if testAuth(req, auth) {
		return true
	}
	denyAuth(res)
	return false
}

func respondJson(resp http.ResponseWriter, data interface{}) {
	b, err := json.Marshal(&data)
	if err != nil {
		resp.WriteHeader(500)
	} else {
		resp.Write(b)
	}
}

func routerHandlerFunc(router *mux.Router) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		router.ServeHTTP(res, req)
	}
}

func static(res http.ResponseWriter, req *http.Request) {
	http.ServeFile(res, req, "public"+req.URL.Path)
}

func notFound(res http.ResponseWriter, req *http.Request) {
	http.ServeFile(res, req, "public/404.html")
}

var herokuAuth Authenticator

type createResourceResp struct {
	Id      string            `json:"id"`
	Config  map[string]string `json:"config"`
	Message string            `json:"message"`
}

func createResource(res http.ResponseWriter, req *http.Request) {
	if !ensureAuth(res, req, herokuAuth) {
		return
	}
	respD := &createResourceResp{
		Id:      "1",
		Config:  map[string]string{"KENSA_CREATE_GO_URL": "https://kensa-create-go.com/resources/1"},
		Message: "All set up!"}
	respondJson(res, &respD)
}

type updateResourceResp struct {
	Config  map[string]string `json:"config"`
	Message string            `json:"message"`
}

func updateResource(resp http.ResponseWriter, req *http.Request) {
	if !ensureAuth(resp, req, herokuAuth) {
		return
	}
	respD := updateResourceResp{
		Config:  map[string]string{"KENSA_CREATE_GO_URL": "https://kensa-create-go.com/resources/1"},
		Message: "All updated!"}
	respondJson(resp, &respD)
}

type destroyResourceResp struct {
	Message string `json:"message"`
}

func destroyResource(res http.ResponseWriter, req *http.Request) {
	if !ensureAuth(res, req, herokuAuth) {
		return
	}
	respD := &destroyResourceResp{
		Message: "All torn down!"}
	respondJson(res, &respD)
}

func router() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/style.css", static).Methods("GET")
	router.HandleFunc("/heroku/resources", createResource).Methods("POST")
	router.HandleFunc("/heroku/resources/{id}", updateResource).Methods("PUT")
	router.HandleFunc("/heroku/resources/{id}", destroyResource).Methods("DELETE")
	// router.HandleFunc("/sso/login", createSession).Methods("POST")
	router.NotFoundHandler = http.HandlerFunc(notFound)
	return router
}

func main() {
	herokuPassword := config("HEROKU_PASSWORD")
	herokuAuth = func(u string, p string) bool {
		return p == herokuPassword
	}
	logs := make(chan string, 10000)
	go runLogging(logs)
	handler := routerHandlerFunc(router())
	handler = wrapLogging(handler, logs)
	port := config("PORT")
	logs <- fmt.Sprintf("serve at=start port=%s", port)
	err := http.ListenAndServe(":"+port, handler)
	check(err)
}

// todo: extract
// todo: constant-time compare
