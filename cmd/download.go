package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	charmLog "github.com/charmbracelet/log"
	"github.com/urfave/cli/v3"

	"github.com/sweepies/tok-dl/db"
	"github.com/sweepies/tok-dl/downloader"
	internalContext "github.com/sweepies/tok-dl/internal/context"
	"github.com/sweepies/tok-dl/internal/regex"
	"github.com/sweepies/tok-dl/tikwm"
	"github.com/sweepies/tok-dl/util"
)

var (
	metadataOnly bool
	outDir       string
	inFile       string
	dbDir        string

	log      *charmLog.Logger
	database *db.Database
)

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
		if !regex.TiktokURL.MatchString(line) {
			continue
		}

		urls = append(urls, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input file: %w", err)
	}

	return urls, nil
}

// saveMetadataFile writes the response data to a JSON file
func saveMetadataFile(data *tikwm.Response, dirPath string) error {
	jsonPath := path.Join(dirPath, "metadata.json")
	jsonData, err := json.MarshalIndent(data.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
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

	// Create directory for the post
	dirPath := path.Join(outDir, data.Data.ID)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Save metadata file
	if err := saveMetadataFile(data, dirPath); err != nil {
		log.Warn("Failed to save metadata file", "err", err)
	}

	if !metadataOnly {
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

// NewDownloadCommand creates the download command
func NewDownloadCommand() *cli.Command {
	return &cli.Command{
		Name:  "download",
		Usage: "Download TikTok videos from URLs in a file",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "metadata-only", Aliases: []string{"m"}, Usage: "only download metadata", Destination: &metadataOnly, Sources: cli.EnvVars("TOKDL_METADATA_ONLY")},
			&cli.StringFlag{Name: "out-dir", Aliases: []string{"o"}, Usage: "output directory", Value: "./tiktok", Destination: &outDir, Sources: cli.EnvVars("TOKDL_OUT_DIR")},
			&cli.StringFlag{Name: "db-dir", Usage: "directory for SQLite database", Destination: &dbDir, DefaultText: "OS user cache dir", Sources: cli.EnvVars("TOKDL_DB_DIR")},
		},
		ArgsUsage: "INPUT_FILE",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Get logger from context
			logger, ok := ctx.Value(internalContext.LoggerKey).(*charmLog.Logger)
			if !ok {
				return ctx, fmt.Errorf("logger not found in context")
			}
			log = logger
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
				return cli.ShowSubcommandHelp(cmd)
			}

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
}
