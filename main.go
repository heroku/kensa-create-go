package main

import (
	"code.google.com/p/gorilla/mux"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// type Authenticator func(string, string) bool
// 
// func testAuth(r *http.Request, auth Authenticator) bool {
// 	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
// 	if len(s) != 2 || s[0] != "Basic" {
// 		return false
// 	}
// 	b, err := base64.StdEncoding.DecodeString(s[1])
// 	if err != nil {
// 		return false
// 	}
// 	pair := strings.SplitN(string(b), ":", 2)
// 	if len(pair) != 2 {
// 		return false
// 	}
// 	return auth(pair[0], pair[1])
// }

// func requireAuth(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("WWW-Authenticate", `Basic realm="private"`)
// 	w.WriteHeader(401)
// 	w.Write([]byte("401 Unauthorized\n"))
// }
// 
// func wrapAuth(h http.HandlerFunc, a Authenticator) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		if testAuth(r, a) {
// 			h(w, r)
// 		} else {
// 			requireAuth(w, r)
// 		}
// 	}
// }

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

// func checkAuth(user, pass string) bool {
// 	auth := os.Getenv("AUTH")
// 	if auth == "" {
// 		return true
// 	}
// 	return auth == strings.Join([]string{user, pass}, ":")
// }

func createResource(res http.ResponseWriter, req *http.Request) {
	// if requireAuth(res, req)
	var respD struct {
		Id string `json:"id"`
	}
	respD.Id = "1"
	respB, err := json.Marshal(&respD)
	if err != nil {
		res.WriteHeader(500)
	} else {
		res.Write(respB)
	}
}

func router() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/style.css", static).Methods("GET")
	router.HandleFunc("/heroku/resources", createResource).Methods("POST")
	// router.HandleFunc("/heroku/resources/:id", updateResource).Methods("PUT")
	// router.HandleFunc("/heroku/resources/:id", destroyResource).Methods("DELETE")
	// router.HandleFunc("/sso/login", createSession).Methods("POST")
	router.NotFoundHandler = http.HandlerFunc(notFound)
	return router
}

func main() {
	// initAuthenticator()
	logs := make(chan string, 10000)
	go runLogging(logs)
	handler := routerHandlerFunc(router())
	handler = wrapLogging(handler, logs)
	logs <- "serve at=start"
	err := http.ListenAndServe(":"+config("PORT"), handler)
	check(err)
}
