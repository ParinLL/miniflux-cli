package miniflux

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/parinll/miniflux-cli/internal/config"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	basicAuth  string
	apiToken   string
	debug      bool
	debugOut   io.Writer
}

type Feed struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	SiteURL  string `json:"site_url"`
	FeedURL  string `json:"feed_url"`
	Category struct {
		Title string `json:"title"`
	} `json:"category"`
	CheckedAt string `json:"checked_at"`
}

type CreateFeedInput struct {
	FeedURL    string `json:"feed_url"`
	CategoryID int64  `json:"category_id,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Crawler    bool   `json:"crawler,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

type UpdateFeedInput struct {
	FeedURL    *string `json:"feed_url,omitempty"`
	SiteURL    *string `json:"site_url,omitempty"`
	Title      *string `json:"title,omitempty"`
	CategoryID *int64  `json:"category_id,omitempty"`
	Username   *string `json:"username,omitempty"`
	Password   *string `json:"password,omitempty"`
	UserAgent  *string `json:"user_agent,omitempty"`
}

type createFeedResponse struct {
	FeedID int64 `json:"feed_id"`
}

type ClientOptions struct {
	Timeout     time.Duration
	Debug       bool
	DebugOutput io.Writer
}

type Entry struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	FeedID      int64  `json:"feed_id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	CommentsURL string `json:"comments_url"`
	Author      string `json:"author"`
	Status      string `json:"status"`
	ReadingTime int64  `json:"reading_time"`
	PublishedAt string `json:"published_at"`
	CreatedAt   string `json:"created_at"`
	ChangedAt   string `json:"changed_at"`
	Content     string `json:"content"`
}

type EntriesResponse struct {
	Total   int64   `json:"total"`
	Entries []Entry `json:"entries"`
}

type EntriesFilter struct {
	Status     string
	Limit      int
	Offset     int
	FeedID     int64
	CategoryID int64
}

func New(cfg config.Config, opts ClientOptions) (*Client, error) {
	if cfg.Token == "" && (cfg.Username == "" || cfg.Password == "") {
		return nil, errors.New("set MINIFLUX_API_TOKEN or both MINIFLUX_USERNAME and MINIFLUX_PASSWORD")
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL = strings.TrimRight(baseURL, "/") + "/v1"
	}

	basicAuth := ""
	apiToken := ""
	if cfg.Token != "" {
		apiToken = cfg.Token
	} else {
		creds := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))
		basicAuth = "Basic " + creds
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
		basicAuth: basicAuth,
		apiToken:  apiToken,
		debug:     opts.Debug,
		debugOut:  opts.DebugOutput,
	}, nil
}

func (c *Client) Health() (string, error) {
	body, err := c.do(http.MethodGet, c.healthURL(), nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func (c *Client) Feeds() ([]Feed, error) {
	var feeds []Feed
	body, err := c.do(http.MethodGet, c.apiURL("/feeds"), nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &feeds); err != nil {
		return nil, err
	}
	return feeds, nil
}

func (c *Client) Feed(feedID int64) (Feed, error) {
	var feed Feed
	body, err := c.do(http.MethodGet, c.apiURL("/feeds/"+strconv.FormatInt(feedID, 10)), nil)
	if err != nil {
		return feed, err
	}
	if err := json.Unmarshal(body, &feed); err != nil {
		return feed, err
	}
	return feed, nil
}

func (c *Client) CreateFeed(input CreateFeedInput) (int64, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return 0, err
	}
	body, err := c.do(http.MethodPost, c.apiURL("/feeds"), payload)
	if err != nil {
		return 0, err
	}
	var response createFeedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, err
	}
	return response.FeedID, nil
}

func (c *Client) UpdateFeed(feedID int64, input UpdateFeedInput) (Feed, error) {
	var feed Feed
	payload, err := json.Marshal(input)
	if err != nil {
		return feed, err
	}
	body, err := c.do(http.MethodPut, c.apiURL("/feeds/"+strconv.FormatInt(feedID, 10)), payload)
	if err != nil {
		return feed, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return c.Feed(feedID)
	}
	if err := json.Unmarshal(body, &feed); err != nil {
		return feed, err
	}
	return feed, nil
}

func (c *Client) DeleteFeed(feedID int64) error {
	_, err := c.do(http.MethodDelete, c.apiURL("/feeds/"+strconv.FormatInt(feedID, 10)), nil)
	return err
}

func (c *Client) RefreshAllFeeds() error {
	_, err := c.do(http.MethodPut, c.apiURL("/feeds/refresh"), nil)
	return err
}

func (c *Client) RefreshFeed(feedID int64) error {
	_, err := c.do(http.MethodPut, c.apiURL("/feeds/"+strconv.FormatInt(feedID, 10)+"/refresh"), nil)
	return err
}

func (c *Client) Entries(filter EntriesFilter) (EntriesResponse, error) {
	var entriesResponse EntriesResponse

	query := url.Values{}
	if filter.Status != "" {
		query.Set("status", filter.Status)
	}
	if filter.FeedID > 0 {
		query.Set("feed_id", strconv.FormatInt(filter.FeedID, 10))
	}
	if filter.CategoryID > 0 {
		query.Set("category_id", strconv.FormatInt(filter.CategoryID, 10))
	}
	if filter.Limit > 0 {
		query.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		query.Set("offset", strconv.Itoa(filter.Offset))
	}

	endpoint := c.apiURL("/entries")
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	body, err := c.do(http.MethodGet, endpoint, nil)
	if err != nil {
		return entriesResponse, err
	}
	if err := json.Unmarshal(body, &entriesResponse); err != nil {
		return entriesResponse, err
	}

	return entriesResponse, nil
}

func (c *Client) Entry(entryID int64) (Entry, error) {
	var entry Entry
	body, err := c.do(http.MethodGet, c.apiURL("/entries/"+strconv.FormatInt(entryID, 10)), nil)
	if err != nil {
		return entry, err
	}
	if err := json.Unmarshal(body, &entry); err != nil {
		return entry, err
	}
	return entry, nil
}

func (c *Client) apiURL(path string) string {
	return c.baseURL + path
}

func (c *Client) healthURL() string {
	return strings.TrimSuffix(c.baseURL, "/v1") + "/healthcheck"
}

func (c *Client) do(method, url string, payload []byte) ([]byte, error) {
	start := time.Now()
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if c.apiToken != "" {
		req.Header.Set("X-Auth-Token", c.apiToken)
	} else {
		req.Header.Set("Authorization", c.basicAuth)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.debugf("%s %s error: %v", method, url, err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.debugf("%s %s read body error: %v", method, url, err)
		return nil, err
	}
	c.debugf("%s %s -> %d (%s)", method, url, resp.StatusCode, time.Since(start))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	return respBody, nil
}

func (c *Client) debugf(format string, args ...any) {
	if !c.debug || c.debugOut == nil {
		return
	}
	_, _ = fmt.Fprintf(c.debugOut, "[debug] "+format+"\n", args...)
}
