package gtfs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
)

type dmfrfile struct {
	Feeds []struct {
		ID   string `json:"id"`
		Spec string `json:"spec"`
		URLs struct {
			StaticCurrent  string   `json:"static_current"`
			StaticHistoric []string `json:"static_historic"`
		} `json:"urls"`
		License struct {
			RedistributionAllowed string `json:"redistribution_allowed"`
		} `json:"license"`
	} `json:"feeds"`
}

func ImportDMFRFolder(location string) ([]GTFS, error) {
	item, err := os.ReadDir(location)
	if err != nil {
		return nil, err
	}
	var feeds []GTFS
	for _, file := range item {
		if !file.IsDir() {
			fn := path.Join(location, file.Name())
			jf, err := os.Open(fn)
			if err != nil {
				return nil, fmt.Errorf("Error while opening the file %v: %v", fn, err)
			}
			defer jf.Close()
			bv, err := io.ReadAll(jf)
			if err != nil {
				return nil, fmt.Errorf("Error while reading the file %v: %v", fn, err)
			}
			var file dmfrfile
			json.Unmarshal(bv, &file)
			for _, newfeed := range file.Feeds {
				if newfeed.Spec != "gtfs" {
					continue
				}
				if len(newfeed.URLs.StaticCurrent) == 0 {
					continue
				}
				if newfeed.License.RedistributionAllowed == "no" {
					continue
				}
				feeds = append(feeds, GTFS{
					ID:        newfeed.ID,
					SourceURL: newfeed.URLs.StaticCurrent,
				})
			}
		}
	}
	return feeds, nil
}
