package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	var vcount int

	programLevel := slog.LevelError
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel}))
	slog.SetDefault(logger)
	app := &cli.App{
		Name:                   "gtfsproxy",
		Usage:                  "a proxy for gtfs files",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "verbosity (-v, -vv, -vvv)",
				Count:   &vcount,
				Action: func(ctx *cli.Context, v bool) error {
					switch vcount {
					case 1:
						programLevel = slog.LevelWarn
					case 2:
						programLevel = slog.LevelInfo
					case 3:
						programLevel = slog.LevelDebug
					}
					logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel}))
					slog.SetDefault(logger)
					return nil
				},
			},
			&cli.StringFlag{
				Name:    "data",
				Aliases: []string{"d"},
				Value:   "data",
				Usage:   "data folder",
				EnvVars: []string{"GTFSPROXY_DATA_FOLDER"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "import-dmfr",
				Usage:  "import-dmfr FOLDER",
				Action: importDMFR,
			},
			{
				Name:   "download",
				Usage:  "download [GTFS]",
				Action: download,
			},
			{
				Name:   "serve",
				Usage:  "serve",
				Action: serve,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "high-ports",
						Usage: "use ports 1080 and 10443 instead of 80 and 443",
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
