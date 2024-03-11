package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/fale/gtfsproxy/pkg/gtfs"
	"github.com/urfave/cli/v2"
)

func importDMFR(ctx *cli.Context) error {
	if len(ctx.Args().First()) == 0 {
		return fmt.Errorf("missing folder")
	}

	if err := os.MkdirAll(ctx.String("data"), os.ModePerm); err != nil {
		return err
	}

	files, err := gtfs.ImportDMFRFolder(ctx.Args().First())
	if err != nil {
		slog.Error(err.Error())
	}

	for _, f := range files {
		d, err := gtfs.Load(ctx.String("data"), f.ID)
		if err != nil {
			err := f.Save(ctx.String("data"))
			if err != nil {
				slog.Error(err.Error())
			}
		}
		if d.SourceURL != f.SourceURL {
			slog.Info("feed updated", "ID", f.ID)
			slog.Debug("feed updated", "ID", f.ID, "previous SourceURL", d.SourceURL, "new SourceURL", f.SourceURL)
			d.SourceURL = f.SourceURL
			err := d.Save("data")
			if err != nil {
				slog.Error(err.Error())
			}
		}
	}
	return nil
}
