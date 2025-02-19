package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/sweepies/tok-dl/db"
	"github.com/sweepies/tok-dl/downloader"
	"github.com/sweepies/tok-dl/tikwm"
	"github.com/sweepies/tok-dl/util"

	charmLog "github.com/charmbracelet/log"
	"github.com/urfave/cli/v3"
)

var (
	metadataOnly bool
	outDir       string
	inFile       string
	debug        bool
	dbDir        string

	log      *charmLog.Logger
	database *db.Database

	tiktokURLRegex = regexp.MustCompile(`^https?:\/\/(?:(?:www|vm|vt|m)\.)?tiktokv?\.com\/.+$`)
)

func configure() error {
	level := charmLog.InfoLevel
	if debug {
		level = charmLog.DebugLevel
	}

	log = charmLog.NewWithOptions(os.Stderr, charmLog.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Level:           level,
	})

	if dbDir == "" {
		var err error
		dbDir, err = os.UserCacheDir()
		if err != nil {
			dbDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("could not determine database directory: %w", err)
			}
		}
	}

	var err error
	database, err = db.New(dbDir)
	if err != nil {
		return fmt.Errorf("could not initialize database: %w", err)
	}

	return nil
}

// processInputFile reads URLs from a file, handling comments and empty lines
func processInputFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening input file: %w", err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}
		if !tiktokURLRegex.MatchString(line) {
			continue
		}

		urls = append(urls, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input file: %w", err)
	}

	return urls, nil
}

// downloadPost handles downloading a single TikTok post
func downloadPost(url string, api *tikwm.Client, dl *downloader.MediaDownloader) error {
	data, err := api.FetchMetadata(url)
	if err != nil {
		if errors.Is(err, tikwm.ErrRateLimit) {
			return fmt.Errorf("rate limit exceeded, stopping: %w", err)
		}
		return fmt.Errorf("API error: %w", err)
	}

	if !metadataOnly {
		dirPath := path.Join(outDir, data.Data.ID)
		if err := os.MkdirAll(dirPath, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}

		// Determine what to download
		var mediaURLs []string
		if len(data.Data.Images) > 0 {
			mediaURLs = data.Data.Images
		} else {
			downloadURL := util.StringNotEmptyCoalesce(data.Data.Hdplay, data.Data.Play, data.Data.Wmplay)
			if downloadURL != "" {
				mediaURLs = []string{downloadURL}
			}
		}

		// Verify we have media URLs
		if len(mediaURLs) == 0 {
			if err := database.MarkFailed(url, db.StatusNoMedia); err != nil {
				log.Error("Failed to mark as no media", "err", err)
			}
			return fmt.Errorf("no media URLs found in response")
		}

		// Download the media
		if err := dl.DownloadMedia(mediaURLs, dirPath); err != nil {
			if err := database.MarkFailed(url, db.StatusDownloadFailed); err != nil {
				log.Error("Failed to mark download as failed", "err", err)
			}
			return fmt.Errorf("download failed: %w", err)
		}
	}

	if err := database.MarkComplete(url); err != nil {
		log.Error("Failed to mark as complete", "err", err)
	}

	log.Info("Post processed", "id", data.Data.ID)
	return nil
}

func main() {
	cmd := &cli.Command{
		Name:  "tok-dl",
		Usage: "A TikTok Downloader that actually works",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "metadata-only", Aliases: []string{"m"}, Usage: "only download metadata", Destination: &metadataOnly, Sources: cli.EnvVars("TOKDL_METADATA_ONLY")},
			&cli.StringFlag{Name: "out-dir", Aliases: []string{"o"}, Usage: "output directory", Value: "./tiktok", Destination: &outDir, Sources: cli.EnvVars("TOKDL_OUT_DIR")},
			&cli.StringFlag{Name: "db-dir", Usage: "directory for SQLite database", Destination: &dbDir, DefaultText: "OS user cache dir", Sources: cli.EnvVars("TOKDL_DB_DIR")},
			&cli.BoolFlag{Name: "debug", Usage: "show debug logs", Destination: &debug, Sources: cli.EnvVars("TOKDL_DEBUG")},
		},
		ArgsUsage: "INPUT_FILE",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if err := configure(); err != nil {
				return ctx, err
			}
			return ctx, nil
		},
		After: func(ctx context.Context, cmd *cli.Command) error {
			if database != nil {
				return database.Close()
			}
			return nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			inFile = cmd.Args().First()
			if inFile == "" {
				cli.ShowAppHelpAndExit(cmd, 1)
			}

			urls, err := processInputFile(inFile)
			if err != nil {
				log.Fatal("Failed to process input file", "err", err)
			}
			log.Info("File loaded", "urls", len(urls))

			if err := os.MkdirAll(outDir, 0700); err != nil {
				log.Fatal("Could not create output directory", "dir", outDir, "err", err)
			}

			api := tikwm.New(database, log)
			defer api.Close()

			dl := downloader.NewMediaDownloader()
			defer dl.Close()

			for _, url := range urls {
				if err := downloadPost(url, api, dl); err != nil {
					if errors.Is(err, tikwm.ErrRateLimit) {
						log.Fatal(err) // Rate limit means we should stop
					}
					log.Warn("Failed to process post", "url", url, "err", err)
					continue
				}
			}

			log.Info("Finished")
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
