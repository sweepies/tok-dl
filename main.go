package main

import (
	"context"
	"os"

	"github.com/sweepies/tok-dl/cmd"
	internalContext "github.com/sweepies/tok-dl/internal/context"
	"github.com/urfave/cli/v3"

	charmLog "github.com/charmbracelet/log"
)

func main() {
	var debug bool

	app := &cli.Command{
		Name:  "tok-dl",
		Usage: "A TikTok Downloader that actually works",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "show debug logs",
				Destination: &debug,
				Sources:     cli.EnvVars("TOKDL_DEBUG"),
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			level := charmLog.InfoLevel
			if debug {
				level = charmLog.DebugLevel
			}

			log := charmLog.NewWithOptions(os.Stderr, charmLog.Options{
				ReportTimestamp: true,
				TimeFormat:      "15:04:05",
				Level:           level,
			})

			return context.WithValue(ctx, internalContext.LoggerKey, log), nil
		},
		Commands: []*cli.Command{
			cmd.NewDownloadCommand(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log := charmLog.NewWithOptions(os.Stderr, charmLog.Options{
			ReportTimestamp: true,
			TimeFormat:      "15:04:05",
		})
		log.Fatal(err)
	}
}