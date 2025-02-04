package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	_cache "github.com/sweepies/tok-dl/cache"
	"github.com/sweepies/tok-dl/tikwm"
	"github.com/sweepies/tok-dl/util"

	charmLog "github.com/charmbracelet/log"
	"github.com/urfave/cli/v3"
)

var (
	metadataOnly bool
	outDir       string
	noCache      bool
	inFile       string
	debug        bool
	cacheDir     string

	log   *charmLog.Logger
	cache *_cache.Cache

	urls []string

	mimeExts = map[string]string{
		"video/mp4":  ".mp4",
		"audio/mpeg": ".mp3",
		"audio/mp3":  ".mp3",
	}
)

func configure() {
	level := charmLog.InfoLevel

	if debug {
		level = charmLog.DebugLevel
	}

	log = charmLog.NewWithOptions(os.Stderr, charmLog.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Level:           level,
	})

	cache = _cache.New(cacheDir)
}

func main() {
	cmd := &cli.Command{
		Name:  "tok-dl",
		Usage: "A TikTok Downloader that actually works",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "metadata-only", Aliases: []string{"m"}, Usage: "only download metadata", Destination: &metadataOnly},
			&cli.StringFlag{Name: "out-dir", Aliases: []string{"o"}, Usage: "output directory", Value: "./tiktok", Destination: &outDir},
			&cli.BoolFlag{Name: "no-cache", Usage: "bypass the cache; don't skip already actioned urls", Destination: &noCache},
			&cli.StringFlag{Name: "cache-dir", Usage: "directory for cache database", Destination: &cacheDir, DefaultText: "OS user cache dir"},
			&cli.BoolFlag{Name: "debug", Usage: "show debug logs", Destination: &debug},
		},
		ArgsUsage: "INPUT_FILE",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			configure()
			return ctx, nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {

			inFile = cmd.Args().First()

			// check if input file is provided
			if len(inFile) == 0 {
				cli.ShowAppHelpAndExit(cmd, 1)
			}

			// read input file
			data, err := os.ReadFile(inFile)
			if err != nil {
				log.Fatal("Error opening input file", "err", err.Error())
			}

			// split by newline, handle carriage returns
			lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

			for _, line := range lines {
				// skip empty lines
				if len(line) == 0 {
					continue
				}

				// ignore comments
				exp := regexp.MustCompile("^(#|//|--)")
				if exp.FindStringIndex(line) != nil {
					continue
				}

				// skip invalid urls
				_, err := url.ParseRequestURI(line)
				if err != nil {
					continue
				}

				urls = append(urls, line)
			}

			log.Info("File loaded", "urls", len(urls))

			err = os.MkdirAll(outDir, 0700)

			if err != nil {
				log.Fatal("Could not create directory", "dir", outDir, "err", err.Error())
			}

			caller := tikwm.New(cache, log)

			for _, url := range urls {
				if !noCache && string(cache.Get([]byte(url))) != "" {
					log.Debug("URL was found in cache; skipping", "url", url)
					continue
				}

				data, err := caller.FetchMetadata(url)

				if err != nil {
					if errors.Is(err, tikwm.ErrRateLimit) {
						log.Fatal("Exiting", "err", err.Error())
					}

					log.Warn("Error from API", "err", err.Error(), "url", url)
					cache.Set([]byte(url), []byte(err.Error()))

					continue
				}

				dirPath := path.Join(outDir, data.Data.ID)
				err = os.MkdirAll(dirPath, 0700)

				if err != nil {
					log.Fatal("Could not create directory", "dir", dirPath, "err", err.Error())
				}

				filePath := path.Join(dirPath, fmt.Sprintf("%s.json", data.Data.ID))

				file, err := os.Create(filePath)
				if err != nil {
					log.Fatal("Could not open file", "file", filePath, "err", err.Error())
				}

				encoder := json.NewEncoder(file)
				encoder.SetIndent("", " ")
				encoder.Encode(data.Data)
				file.Close()

				// download files
				if len(data.Data.Images) > 0 {

					var imageErrs []error
					for _, image := range data.Data.Images {
						err := downloadFileInferExt(image, dirPath)

						if err != nil {
							log.Warn("Error downloading image", "err", err.Error())
							imageErrs = append(imageErrs, err)
						}
					}
					if len(imageErrs) > 0 {
						log.Warn("Download incomplete", "id", data.Data.ID)
						cache.Set([]byte(url), []byte("image(s) had errs"))
					}
				} else {
					downloadUrl := util.StringNotEmptyCoalesce(data.Data.Hdplay, data.Data.Play, data.Data.Wmplay)

					err := downloadFileInferExt(downloadUrl, dirPath)

					if err != nil {
						log.Warn("Error downloading video", "err", err.Error())
						cache.Set([]byte(url), []byte(err.Error()))

						continue
					}

				}

				// finished
				log.Info("Download finished", "id", data.Data.ID)
				cache.Set([]byte(url), []byte("satisfied"))
			}

			log.Info("Finished")
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func downloadFileInferExt(downloadUrl string, destDir string) error {
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return errors.New(resp.Status)
	}

	parsedUrl, _ := url.Parse(downloadUrl)
	fileName := util.SanitizeFileName(filepath.Base(parsedUrl.Path))

	if filepath.Ext(fileName) == "" {
		contentType := resp.Header.Get("Content-Type")

		var ext string

		if mimeExts[contentType] != "" {
			ext = mimeExts[contentType]
		} else {
			exts, _ := mime.ExtensionsByType(contentType)
			ext = exts[0]
		}

		fileName = fmt.Sprintf("%s%s", fileName, ext)
	}

	filePath := path.Join(destDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal("Could not open file", "file", filePath)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)

	return err
}
