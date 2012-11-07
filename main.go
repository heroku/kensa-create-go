package main

import (
	"code.google.com/p/gorilla/mux"
 	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func MustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing " + k)
	}
	return v
}

func Sha1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
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
	res.Write([]byte("{message: \"Unauthorized\"}\n"))
}

func ensureAuth(res http.ResponseWriter, req *http.Request, auth Authenticator) bool {
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

type createResourceReq struct {
	HerokuId    string            `json:"heroku_id"`
	Plan        string            `json:"plan"`
	CallbackUrl string            `json:"callback_url"`
	Options     map[string]string `json:"options"`
}
type createResourceResp struct {
	Id      string            `json:"id"`
	Config  map[string]string `json:"config"`
	Message string            `json:"message"`
}

func createResource(resp http.ResponseWriter, req *http.Request) {
	if !ensureAuth(resp, req, herokuAuth) {
		return
	}
	reqD := &createResourceReq{}
	if !readJson(resp, req, reqD) {
		return
	}
	respD := &createResourceResp{
		Id:      "1",
		Config:  map[string]string{"KENSA_CREATE_GO_URL": "https://kensa-create-go.com/resources/1"},
		Message: "All set up!"}
	writeJson(resp, respD)
}

type updateResourceReq struct {
	HerokuId string `json:"heroku_id"`
	Plan     string `json:"plan"`
}
type updateResourceResp struct {
	Config  map[string]string `json:"config"`
	Message string            `json:"message"`
}

func updateResource(resp http.ResponseWriter, req *http.Request) {
	if !ensureAuth(resp, req, herokuAuth) {
		return
	}
	reqD := &updateResourceReq{}
	if !readJson(resp, req, reqD) {
		return
	}
	respD := &updateResourceResp{
		Config:  map[string]string{"KENSA_CREATE_GO_URL": "https://kensa-create-go.com/resources/1"},
		Message: "All updated!"}
	writeJson(resp, respD)
}

type destroyResourceResp struct {
	Message string `json:"message"`
}

func destroyResource(resp http.ResponseWriter, req *http.Request) {
	if !ensureAuth(resp, req, herokuAuth) {
		return
	}
	respD := &destroyResourceResp{
		Message: "All torn down!"}
	writeJson(resp, &respD)
}

func createSession(resp http.ResponseWriter, req *http.Request) {
	readForm(resp, req)
	ssoSalt := MustGetenv("HEROKU_SSO_SALT")
	id := req.FormValue("id")
	timestamp := req.FormValue("timestamp")
	token := req.FormValue("token")
	navData := req.FormValue("nav-data")
	hash := Sha1(id + ":" + ssoSalt + ":" + timestamp)
	if hash != token {
		resp.WriteHeader(403)
		resp.Write([]byte("{message: \"Invalid token\"}"))
		return
	}
	timestampLimit := int(time.Now().Unix() - (2 * 60))
	timestampInt, err := strconv.Atoi(timestamp)
	if (err != nil) || (timestampInt < timestampLimit) {
		resp.WriteHeader(403)
		resp.Write([]byte("{message: \"Invalid timestamp\"}"))
		return
	}
	http.SetCookie(resp, &http.Cookie{
		Name: "heroku-nav-data",
		Value: navData})
	http.Redirect(resp, req, "/", 302)
}

func router() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", static).Methods("GET")
	router.HandleFunc("/style.css", static).Methods("GET")
	router.HandleFunc("/heroku/resources", createResource).Methods("POST")
	router.HandleFunc("/heroku/resources/{id}", updateResource).Methods("PUT")
	router.HandleFunc("/heroku/resources/{id}", destroyResource).Methods("DELETE")
	router.HandleFunc("/sso/login", createSession).Methods("POST")
	router.NotFoundHandler = http.HandlerFunc(notFound)
	return router
}

func main() {
	logs := make(chan string, 10000)
	go runLogging(logs)

	herokuPassword := MustGetenv("HEROKU_PASSWORD")
	herokuAuth = func(u string, p string) bool {
		return p == herokuPassword
	}

	handler := routerHandlerFunc(router())
	handler = wrapLogging(handler, logs)

	port := MustGetenv("PORT")
	logs <- fmt.Sprintf("serve at=start port=%s", port)
	err := http.ListenAndServe(":"+port, handler)
	Check(err)
}

// todo: extract
// todo: constant-time compare
