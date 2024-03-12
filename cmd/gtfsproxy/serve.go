package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"time"

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
	errs := Run(n, ctx.Bool("high-ports"))

	select {
	case err := <-errs:
		log.Printf("Could not start serving service due to (error: %s)", err)
	}
	return nil
}

func Run(h http.Handler, highPorts bool) chan error {
	errs := make(chan error)

	httpPort := ":80"
	httpsPort := ":443"
	if highPorts {
		httpPort = ":1080"
		httpsPort = ":10443"
	}
	go func() {
		log.Printf("Staring HTTP service on %s ...\n", httpPort)

		if err := http.ListenAndServe(httpPort, h); err != nil {
			errs <- err
		}
	}()

	go func() {
		log.Printf("Staring HTTPS service on %s ...\n", httpsPort)
		if err := http.ListenAndServeTLS(httpsPort, "tls.crt", "tls.key", h); err != nil {
			errs <- err
		}
	}()

	return errs
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