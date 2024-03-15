package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/fale/gtfsproxy/pkg/gtfs"
	"github.com/gorilla/mux"
	"github.com/urfave/cli/v2"
	"github.com/urfave/negroni"
)

//go:embed docs.html
var static embed.FS

func serve(ctx *cli.Context) error {
	r := mux.NewRouter()

	r.HandleFunc("/", srvHome).Methods("GET")
	r.HandleFunc("/docs", srvDocs).Methods("GET")
	r.HandleFunc("/{gtfs_id}", srvGTFSHead).Methods("HEAD")
	r.HandleFunc("/{gtfs_id}", srvGTFS).Methods("GET")

	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }).Methods("GET")
	r.HandleFunc("/readiness", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }).Methods("GET")

	n := negroni.New()
	n.Use(negroni.NewLogger())
	n.UseHandler(r)

	log.Println("Ready to serve")
	if len(ctx.String("domain")) > 0 {
		return certmagic.HTTPS([]string{ctx.String("domain")}, n)
	}

	httpPort := ":80"
	if ctx.Bool("high-ports") {
		httpPort = ":1080"
	}
	return http.ListenAndServe(httpPort, n)
}

func srvHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs", http.StatusTemporaryRedirect)
}

func srvDocs(w http.ResponseWriter, r *http.Request) {
	p, err := static.ReadFile("docs.html")
	if err != nil {
		w.Header().Set("X-Error", err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Write(p)
}

func srvGTFSHead(w http.ResponseWriter, r *http.Request) {
	g, err := gtfs.Load("data", mux.Vars(r)["gtfs_id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Last-Modified", g.LastModified.Format(time.RFC1123))
	w.Header().Set("Digest", fmt.Sprintf("sha-256=%s", g.Digest.SHA256))
	w.Header().Set("Content-Type", "application/zip")
	w.WriteHeader(http.StatusOK)
}

func srvGTFS(w http.ResponseWriter, r *http.Request) {
	g, err := gtfs.Load("data", mux.Vars(r)["gtfs_id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Last-Modified", g.LastModified.Format(time.RFC1123))
	w.Header().Set("Digest", fmt.Sprintf("sha-256=%s", g.Digest.SHA256))
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, fmt.Sprintf("data/%s.zip", g.ID))
}
