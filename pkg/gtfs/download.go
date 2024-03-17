package gtfs

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/jlaffaye/ftp"
)

func (g *GTFS) Download(maxAge time.Duration, force bool) error {
	now := time.Now()

	if time.Since(g.LastSuccessfulDownload) <= maxAge {
		slog.Debug(
			fmt.Sprintf("%v skipped, not old enough", g.ID),
			"last successful download", g.LastSuccessfulDownload,
			"max age", maxAge,
		)
		return nil
	}

	slog.Info(fmt.Sprintf("download %v", g.SourceURL))
	u, err := url.Parse(g.SourceURL)
	if err != nil {
		return err
	}
	var body []byte
	switch u.Scheme {
	case "ftp":
		body, err = downloadFTP(g.SourceURL, g.LastModified)
	case "http", "https":
		body, err = downloadHTTP(g.SourceURL, g.LastModified, force, g.InsecureDownload)
	}
	if err != nil {
		g.LastDownloadAttempted = now
		g.Save(g.location)
		return err
	}

	if body == nil {
		slog.Debug(
			fmt.Sprintf("%v not saved, already up to date", g.ID),
			"sha256", g.Digest.SHA256,
		)
		g.LastSuccessfulDownload = now
		g.LastDownloadAttempted = now
		g.Save(g.location)
		return nil
	}

	if contentType := http.DetectContentType(body[:512]); contentType != "application/zip" {
		slog.Debug(
			fmt.Sprintf("%v not saved, since it is not a zip file", g.ID),
			"content type", contentType,
		)
		g.LastDownloadAttempted = now
		g.Save(g.location)
		return nil
	}

	sum256 := sha256.Sum256(body)
	if base64.URLEncoding.EncodeToString(sum256[:]) == g.Digest.SHA256 {
		slog.Debug(
			fmt.Sprintf("%v not saved, already up to date", g.ID),
			"sha256", g.Digest.SHA256,
		)
		g.LastSuccessfulDownload = now
		g.LastDownloadAttempted = now
		g.Save(g.location)
		return nil
	}

	if err := os.WriteFile(path.Join(g.location, fmt.Sprintf("%v.zip", g.ID)), body, 0o644); err != nil {
		return err
	}
	g.Digest.SHA256 = base64.URLEncoding.EncodeToString(sum256[:])
	g.LastModified = now
	g.LastSuccessfulDownload = now
	g.LastDownloadAttempted = now
	g.Save(g.location)

	return nil
}

func downloadHTTP(uri string, lastModified time.Time, force bool, insecureDownload bool) ([]byte, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GTFS Proxy/0.1")
	if !force {
		req.Header.Set("If-Modified-Since", lastModified.Format(time.RFC1123))
	}

	tr := &http.Transport{}
	if insecureDownload {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	c := http.Client{Transport: tr, Timeout: time.Duration(10) * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	}
	return io.ReadAll(resp.Body)
}

func downloadFTP(uri string, lastModified time.Time) ([]byte, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	c, err := ftp.Dial(fmt.Sprintf("%s:21", u.Host), ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}

	user := "anonymous"
	pass := "anonymous"
	if u.User != nil {
		user = u.User.Username()
		if p, ok := u.User.Password(); ok {
			pass = p
		}
	}
	err = c.Login(user, pass)
	if err != nil {
		return nil, err
	}

	if c.IsGetTimeSupported() {
		// Implement https://pkg.go.dev/github.com/jlaffaye/ftp#ServerConn.GetTime
		t, err := c.GetTime(u.Path)
		if err == nil && t.Before(lastModified) {
			return nil, nil
		}
	}

	r, err := c.Retr(u.Path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	buf, err := io.ReadAll(r)
	return buf, err
}
