package gtfs

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"time"
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
	req, err := http.NewRequest("GET", g.SourceURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "GTFS Proxy/0.1")
	req.Header.Set("If-Modified-Since", g.LastModified.Format(time.RFC1123))

	c := http.Client{Timeout: time.Duration(10) * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		g.LastDownloadAttempted = now
		g.Save(g.location)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

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

	if contentType := http.DetectContentType(body[:512]); contentType != "application/zip" {
		slog.Debug(
			fmt.Sprintf("%v not saved, since it is not a zip file", g.ID),
			"content type", contentType,
		)
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
