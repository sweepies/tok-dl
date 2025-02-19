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

// downloadCommand holds the state for the download command
type downloadCommand struct {
	// Flag values
	metadataOnly bool
	outDir       string
	dbDir        string

	// Runtime state
	log      *charmLog.Logger
	database *db.Database
}

// saveMetadataFile writes the response data to a JSON file
func (c *downloadCommand) saveMetadataFile(data *tikwm.Response, dirPath string) error {
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

// processInputFile reads URLs from a file, handling comments and empty lines
func (c *downloadCommand) processInputFile(filename string) ([]string, error) {
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

// downloadPost handles downloading a single TikTok post
func (c *downloadCommand) downloadPost(url string, api *tikwm.Client, dl *downloader.MediaDownloader) error {
	data, err := api.FetchMetadata(url)
	if err != nil {
		if errors.Is(err, tikwm.ErrRateLimit) {
			return fmt.Errorf("rate limit exceeded, stopping: %w", err)
		}
		return fmt.Errorf("API error: %w", err)
	}

	// Create directory for the post
	dirPath := path.Join(c.outDir, data.Data.ID)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Save metadata file
	if err := c.saveMetadataFile(data, dirPath); err != nil {
		c.log.Warn("Failed to save metadata file", "err", err)
	}

	if !c.metadataOnly {
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
			if err := c.database.MarkFailed(url, db.StatusNoMedia); err != nil {
				c.log.Error("Failed to mark as no media", "err", err)
			}
			return fmt.Errorf("no media URLs found in response")
		}

		// Download the media
		if err := dl.DownloadMedia(mediaURLs, dirPath); err != nil {
			if err := c.database.MarkFailed(url, db.StatusDownloadFailed); err != nil {
				c.log.Error("Failed to mark download as failed", "err", err)
			}
			return fmt.Errorf("download failed: %w", err)
		}
	}

	if err := c.database.MarkComplete(url); err != nil {
		c.log.Error("Failed to mark as complete", "err", err)
	}

	c.log.Info("Post processed", "id", data.Data.ID)
	return nil
}

// execute handles the command execution
func (c *downloadCommand) execute(ctx context.Context, inputFile string) error {
	if c.dbDir == "" {
		var err error
		c.dbDir, err = os.UserCacheDir()
		if err != nil {
			c.dbDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("could not determine database directory: %w", err)
			}
		}
	}

	var err error
	c.database, err = db.New(c.dbDir)
	if err != nil {
		return fmt.Errorf("could not initialize database: %w", err)
	}
	defer c.database.Close()

	urls, err := c.processInputFile(inputFile)
	if err != nil {
		c.log.Fatal("Failed to process input file", "err", err)
	}
	c.log.Info("File loaded", "urls", len(urls))

	if err := os.MkdirAll(c.outDir, 0700); err != nil {
		c.log.Fatal("Could not create output directory", "dir", c.outDir, "err", err)
	}

	api := tikwm.New(c.database, c.log)
	defer api.Close()

	dl := downloader.NewMediaDownloader()
	defer dl.Close()

	for _, url := range urls {
		if err := c.downloadPost(url, api, dl); err != nil {
			if errors.Is(err, tikwm.ErrRateLimit) {
				c.log.Fatal(err) // Rate limit means we should stop
			}
			c.log.Warn("Failed to process post", "url", url, "err", err)
			continue
		}
	}

	c.log.Info("Finished")
	return nil
}

// NewDownloadCommand creates the download command
func NewDownloadCommand() *cli.Command {
	cmd := &downloadCommand{
		outDir: "./tiktok", // default value
	}

	return &cli.Command{
		Name:  "download",
		Usage: "Download TikTok videos from URLs in a file",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "metadata-only", Aliases: []string{"m"}, Usage: "only download metadata", Destination: &cmd.metadataOnly, Sources: cli.EnvVars("TOKDL_METADATA_ONLY")},
			&cli.StringFlag{Name: "out-dir", Aliases: []string{"o"}, Usage: "output directory", Value: cmd.outDir, Destination: &cmd.outDir, Sources: cli.EnvVars("TOKDL_OUT_DIR")},
			&cli.StringFlag{Name: "db-dir", Usage: "directory for SQLite database", Destination: &cmd.dbDir, DefaultText: "OS user cache dir", Sources: cli.EnvVars("TOKDL_DB_DIR")},
		},
		ArgsUsage: "INPUT_FILE",
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			// Get logger from context
			logger, ok := ctx.Value(internalContext.LoggerKey).(*charmLog.Logger)
			if !ok {
				return ctx, fmt.Errorf("logger not found in context")
			}
			cmd.log = logger
			return ctx, nil
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			inputFile := c.Args().First()
			if inputFile == "" {
				return cli.ShowSubcommandHelp(c)
			}
			return cmd.execute(ctx, inputFile)
		},
	}
}
