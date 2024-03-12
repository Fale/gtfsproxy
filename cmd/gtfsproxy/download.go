package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/fale/gtfsproxy/pkg/gtfs"
	"github.com/urfave/cli/v2"
)

func download(ctx *cli.Context) error {
	var ds []gtfs.GTFS
	var err error
	if ctx.NArg() > 0 {
		d, err := gtfs.Load(ctx.String("data"), ctx.Args().First())
		if err != nil {
			slog.Error(err.Error())
		}
		ds = append(ds, d)
	} else {
		ds, err = gtfs.LoadAll(ctx.String("data"))
		if err != nil {
			slog.Error(err.Error())
		}
	}

	maxAge, _ := time.ParseDuration("24h")

	for _, d := range ds {
		slog.Debug(fmt.Sprintf("start downloading %v", d.ID))
		if err := d.Download(maxAge, false); err != nil {
			slog.Error(err.Error())
		}
	}
	return nil
}
