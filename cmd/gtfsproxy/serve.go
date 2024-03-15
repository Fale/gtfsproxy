package main

import (
	"embed"
	"fmt"
	"html/template"
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
var docsFile embed.FS

//go:embed stats.tpl.html
var statsTpl embed.FS

func serve(ctx *cli.Context) error {
	r := mux.NewRouter()

	r.HandleFunc("/", srvHome).Methods("GET")
	r.HandleFunc("/docs", srvDocs).Methods("GET")
	r.HandleFunc("/stats", srvStats).Methods("GET")
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
	p, err := docsFile.ReadFile("docs.html")
	if err != nil {
		w.Header().Set("X-Error", err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Write(p)
}

func srvStats(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("stats.tpl.html").ParseFS(statsTpl, "stats.tpl.html"))
	d := struct {
		NeverSuccessful []gtfs.GTFS
		RecentlyFailed  []gtfs.GTFS
		Successful      []gtfs.GTFS
	}{}
	gs, err := gtfs.LoadAll("data")
	if err != nil {
		w.Header().Set("X-Error", err.Error())
		return
	}
	for _, g := range gs {
		if g.LastSuccessfulDownload.IsZero() {
			d.NeverSuccessful = append(d.NeverSuccessful, g)
			continue
		}
		// Older than 72hrs
		if time.Since(g.LastSuccessfulDownload) < time.Duration(259200000000000) {
			d.RecentlyFailed = append(d.RecentlyFailed, g)
			continue
		}
		d.Successful = append(d.Successful, g)
	}
	if err := t.Execute(w, d); err != nil {
		w.Header().Set("X-Error", err.Error())
		return
	}
	return
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
