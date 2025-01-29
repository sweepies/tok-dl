package tikwm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	netUrl "net/url"

	"github.com/charmbracelet/log"
	_cache "github.com/sweepies/tok-dl/cache"
)

const (
	BaseUrl = "https://tikwm.com/api/"
)

type ApiCaller struct {
	cache  *_cache.Cache
	client *http.Client
	log    *log.Logger
}

var (
	ErrRateLimit = errors.New("rate limit exceeded")
	ErrParse     = errors.New("parse error")
	ErrUnknown   = errors.New("unknown error")
)

func New(cache *_cache.Cache, logger *log.Logger) ApiCaller {
	caller := &ApiCaller{
		cache: cache,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		log: logger,
	}

	return *caller
}

func (c *ApiCaller) FetchMetadata(postUrl string) (ApiResponse, error) {
	postUrl = fmt.Sprintf("%s?url=%s", BaseUrl, netUrl.QueryEscape(postUrl))
	req, _ := http.NewRequest("GET", postUrl, nil)
	req.Header.Set("Accept", "application/json")

	stamp := c.cache.GetTimestamp()

	if !stamp.IsZero() {
		time.Sleep(time.Until(stamp.Add(1 * time.Second)))
	}

	resp, err := c.client.Do(req)

	c.cache.WriteTimestamp()

	if err != nil {
		return ApiResponse{}, err
	}
	defer resp.Body.Close()

	var data ApiResponse
	json.NewDecoder(resp.Body).Decode(&data)

	if data.Code != 0 {
		switch {
		case strings.HasPrefix(data.Msg, "Free Api Limit"):
			return data, ErrRateLimit
		case strings.HasPrefix(data.Msg, "Url parsing is failed"):
			return data, ErrParse
		default:
			return data, ErrUnknown
		}
	}

	return data, nil
}
