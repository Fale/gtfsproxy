package gtfs

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type GTFS struct {
	ID        string
	SourceURL string
	Digest    struct {
		SHA256 string
	}
	LastModified           time.Time
	LastDownloadAttempted  time.Time
	LastSuccessfulDownload time.Time
	location               string
	InsecureDownload       bool
}

func Load(location string, name string) (GTFS, error) {
	fn := path.Join(location, fmt.Sprintf("%v.json", name))
	var data GTFS

	jf, err := os.Open(fn)
	if err != nil {
		return data, fmt.Errorf("Error while opening the file %v: %v", fn, err)
	}
	defer jf.Close()
	body, err := io.ReadAll(jf)
	if err != nil {
		return data, fmt.Errorf("Error while reading the file %v: %v", fn, err)
	}
	json.Unmarshal(body, &data)
	data.location = location
	return data, nil
}

func LoadAll(location string) ([]GTFS, error) {
	slog.Debug(fmt.Sprintf("read files from %v", location))
	items, err := os.ReadDir(location)
	if err != nil {
		return nil, err
	}
	var feeds []GTFS
	for _, file := range items {
		if filepath.Ext(file.Name()) != ".json" {
			slog.Debug(fmt.Sprintf("ignore file %v", file.Name()))
			continue
		}
		name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		slog.Debug(fmt.Sprintf("read file %v", name))
		feed, err := Load(location, name)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	return feeds, nil
}

func (g *GTFS) Save(location string) error {
	file, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(location, fmt.Sprintf("%v.json", g.ID)), file, 0o644)
}
