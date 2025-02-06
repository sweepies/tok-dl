package tikwm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	netUrl "net/url"

	"github.com/sweepies/tok-dl/db"
	"github.com/charmbracelet/log"
)

const (
	BaseUrl = "https://tikwm.com/api/"
)

type Cache interface {
	IsProcessed(url string) (bool, string, error)
	MarkComplete(url string) error
	MarkFailed(url string, status string) error
	SaveMetadata(id string, url string, data interface{}) error
}

type Client struct {
	cache       Cache
	client      *http.Client
	log         *log.Logger
	lastRequest time.Time
}

var (
	ErrRateLimit = errors.New("rate limit exceeded")
	ErrParse     = errors.New("parse error")
	ErrUnknown   = errors.New("unknown error")
)

func New(cache Cache, logger *log.Logger) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxConnsPerHost = 100
	transport.MaxIdleConnsPerHost = 100

	return &Client{
		cache: cache,
		client: &http.Client{
			Timeout:   time.Second * 10,
			Transport: transport,
		},
		log: logger,
	}
}

func (c *Client) FetchMetadata(postUrl string) (*Response, error) {
	c.log.Debug("Fetching metadata", "url", postUrl)

	// Simple rate limiting - wait if needed
	if !c.lastRequest.IsZero() {
		waitTime := time.Until(c.lastRequest.Add(time.Second))
		if waitTime > 0 {
			time.Sleep(waitTime)
		}
	}

	apiUrl := fmt.Sprintf("%s?url=%s", BaseUrl, netUrl.QueryEscape(postUrl))
	req, _ := http.NewRequest("GET", apiUrl, nil)
	req.Header.Set("Accept", "application/json")

	c.lastRequest = time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Code.Bool() {
		var err error
		var status string

		switch {
		case strings.HasPrefix(response.Msg, "Free Api Limit"):
			err = ErrRateLimit
			status = db.StatusRateLimited
		case strings.HasPrefix(response.Msg, "Url parsing is failed"):
			err = ErrParse
			status = db.StatusParseFailed
		default:
			err = ErrUnknown
			status = db.StatusUnknownError
		}

		// Cache the error state
		if cacheErr := c.cache.MarkFailed(postUrl, status); cacheErr != nil {
			c.log.Warn("Failed to cache error status", "url", postUrl, "status", status, "err", cacheErr)
		}

		return nil, err
	}

	// Save the metadata
	if err := c.cache.SaveMetadata(response.Data.ID, postUrl, response.Data); err != nil {
		c.log.Warn("Failed to save metadata", "id", response.Data.ID, "err", err)
	}

	return &response, nil
}

// Close cleans up any resources used by the client
func (c *Client) Close() error {
	c.client.CloseIdleConnections()
	return nil
}