package main

import (
	"code.google.com/p/gorilla/mux"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

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

var herokuAuth authenticator

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
		Config:  map[string]string{"MYADDON_URL": "https://kensa-create-go.com/resources/1"},
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
		Config:  map[string]string{"MYADDON_URL": "https://kensa-create-go.com/resources/1"},
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
	ssoSalt := mustGetenv("SSO_SALT")
	id := req.FormValue("id")
	timestamp := req.FormValue("timestamp")
	token := req.FormValue("token")
	navData := req.FormValue("nav-data")
	hash := sha1String(id + ":" + ssoSalt + ":" + timestamp)
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

	herokuPassword := mustGetenv("HEROKU_PASSWORD")
	herokuAuth = func(u string, p string) bool {
		return p == herokuPassword
	}

	handler := routerHandlerFunc(router())
	handler = wrapLogging(handler, logs)

	port := mustGetenv("PORT")
	logs <- fmt.Sprintf("serve at=start port=%s", port)
	err := http.ListenAndServe(":"+port, handler)
	check(err)
}
