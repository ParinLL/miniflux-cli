package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/parinll/miniflux-cli/internal/config"
	"github.com/parinll/miniflux-cli/internal/miniflux"
)

const defaultTimeout = 15 * time.Second

var ErrUsage = errors.New("usage error")

func Run(args []string, stdout io.Writer, stderr io.Writer) error {
	cfg := config.FromEnv()

	fs := flag.NewFlagSet("miniflux-cli", flag.ContinueOnError)
	fs.SetOutput(stderr)

	baseURL := fs.String("base-url", config.DefaultBaseURL, "Miniflux API base URL")
	username := fs.String("username", "", "Miniflux username")
	password := fs.String("password", "", "Miniflux password")
	token := fs.String("token", "", "Miniflux API token")
	timeout := fs.Duration("timeout", defaultTimeout, "HTTP timeout")
	debug := fs.Bool("debug", false, "Enable debug logs to stderr")

	fs.Usage = func() {
		usage(fs, stderr)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		usage(fs, stderr)
		return ErrUsage
	}

	isSet := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		isSet[f.Name] = true
	})

	resolved := config.Config{
		BaseURL:  *baseURL,
		Username: *username,
		Password: *password,
		Token:    *token,
	}
	if !isSet["base-url"] {
		resolved.BaseURL = cfg.BaseURL
	}
	if !isSet["username"] {
		resolved.Username = cfg.Username
	}
	if !isSet["password"] {
		resolved.Password = cfg.Password
	}
	if !isSet["token"] {
		resolved.Token = cfg.Token
	}

	client, err := miniflux.New(resolved, miniflux.ClientOptions{
		Timeout:     *timeout,
		Debug:       *debug,
		DebugOutput: stderr,
	})
	if err != nil {
		return err
	}

	command := fs.Arg(0)
	cmdArgs := fs.Args()[1:]

	switch command {
	case "health":
		health, err := client.Health()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, health)
		return err
	case "feeds":
		if len(cmdArgs) == 1 && cmdArgs[0] == "refresh" {
			if err := client.RefreshAllFeeds(); err != nil {
				return err
			}
			_, err := fmt.Fprintln(stdout, `{"refreshed":"all"}`)
			return err
		}
		if len(cmdArgs) > 0 {
			_, _ = fmt.Fprintln(stderr, "usage: miniflux-cli feeds")
			_, _ = fmt.Fprintln(stderr, "or:    miniflux-cli feeds refresh")
			return ErrUsage
		}
		feeds, err := client.Feeds()
		if err != nil {
			return err
		}
		return printJSON(stdout, feeds)
	case "feed":
		return runFeedCommand(client, cmdArgs, stdout, stderr)
	case "entries":
		entriesFlags := flag.NewFlagSet("entries", flag.ContinueOnError)
		entriesFlags.SetOutput(stderr)
		status := entriesFlags.String("status", "unread", "Entry status: unread|read|removed")
		limit := entriesFlags.Int("limit", 20, "Maximum number of entries")
		offset := entriesFlags.Int("offset", 0, "Result offset")
		feedID := entriesFlags.Int64("feed-id", 0, "Filter by feed ID")
		categoryID := entriesFlags.Int64("category-id", 0, "Filter by category ID")
		if err := entriesFlags.Parse(cmdArgs); err != nil {
			return err
		}
		result, err := client.Entries(miniflux.EntriesFilter{
			Status:     *status,
			Limit:      *limit,
			Offset:     *offset,
			FeedID:     *feedID,
			CategoryID: *categoryID,
		})
		if err != nil {
			return err
		}
		return printJSON(stdout, result)
	case "entry":
		if len(cmdArgs) != 1 {
			_, _ = fmt.Fprintln(stderr, "usage: miniflux-cli entry <entry-id>")
			return ErrUsage
		}
		entryID, err := strconv.ParseInt(cmdArgs[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid entry id %q", cmdArgs[0])
		}
		entry, err := client.Entry(entryID)
		if err != nil {
			return err
		}
		return printJSON(stdout, entry)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func usage(fs *flag.FlagSet, w io.Writer) {
	_, _ = fmt.Fprintf(w, `miniflux-cli talks to the Miniflux HTTP API.

Usage:
  miniflux-cli [flags] <command> [args]

Commands:
  health                 GET /healthcheck
  feeds                  GET /v1/feeds
  feeds refresh          PUT /v1/feeds/refresh
  feed list              GET /v1/feeds
  feed get <feed-id>     GET /v1/feeds/{feed-id}
  feed create            POST /v1/feeds
  feed update <feed-id>  PUT /v1/feeds/{feed-id}
  feed delete <feed-id>  DELETE /v1/feeds/{feed-id}
  feed refresh <feed-id> PUT /v1/feeds/{feed-id}/refresh
  entries                GET /v1/entries
  entry <entry-id>       GET /v1/entries/{entry-id}

Flags:
`)
	fs.PrintDefaults()
	_, _ = fmt.Fprintf(w, `
Environment variables:
  %s
  %s
  %s
  %s
`, config.EnvBaseURL, config.EnvUsername, config.EnvPassword, config.EnvToken)
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func runFeedCommand(client *miniflux.Client, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: miniflux-cli feed <list|get|create|update|delete> ...")
		return ErrUsage
	}

	switch args[0] {
	case "list":
		feeds, err := client.Feeds()
		if err != nil {
			return err
		}
		return printJSON(stdout, feeds)
	case "get":
		if len(args) != 2 {
			_, _ = fmt.Fprintln(stderr, "usage: miniflux-cli feed get <feed-id>")
			return ErrUsage
		}
		feedID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid feed id %q", args[1])
		}
		feed, err := client.Feed(feedID)
		if err != nil {
			return err
		}
		return printJSON(stdout, feed)
	case "create":
		createFlags := flag.NewFlagSet("feed create", flag.ContinueOnError)
		createFlags.SetOutput(stderr)
		feedURL := createFlags.String("feed-url", "", "Feed URL (required)")
		categoryID := createFlags.Int64("category-id", 0, "Category ID")
		username := createFlags.String("feed-username", "", "Feed username for authenticated feeds")
		password := createFlags.String("feed-password", "", "Feed password for authenticated feeds")
		crawler := createFlags.Bool("crawler", false, "Force full crawler for this feed")
		userAgent := createFlags.String("user-agent", "", "Custom user agent for this feed")
		if err := createFlags.Parse(args[1:]); err != nil {
			return err
		}
		if *feedURL == "" {
			_, _ = fmt.Fprintln(stderr, "usage: miniflux-cli feed create --feed-url <url> [--category-id <id>]")
			return ErrUsage
		}
		createdID, err := client.CreateFeed(miniflux.CreateFeedInput{
			FeedURL:    *feedURL,
			CategoryID: *categoryID,
			Username:   *username,
			Password:   *password,
			Crawler:    *crawler,
			UserAgent:  *userAgent,
		})
		if err != nil {
			return err
		}
		return printJSON(stdout, map[string]int64{"feed_id": createdID})
	case "update":
		updateFlags := flag.NewFlagSet("feed update", flag.ContinueOnError)
		updateFlags.SetOutput(stderr)
		feedURL := updateFlags.String("feed-url", "", "Update feed URL")
		siteURL := updateFlags.String("site-url", "", "Update site URL")
		title := updateFlags.String("title", "", "Update title")
		categoryID := updateFlags.Int64("category-id", 0, "Update category ID")
		username := updateFlags.String("feed-username", "", "Update feed username")
		password := updateFlags.String("feed-password", "", "Update feed password")
		userAgent := updateFlags.String("user-agent", "", "Update user agent")
		if err := updateFlags.Parse(args[1:]); err != nil {
			return err
		}
		if updateFlags.NArg() != 1 {
			_, _ = fmt.Fprintln(stderr, "usage: miniflux-cli feed update [flags] <feed-id>")
			return ErrUsage
		}
		feedID, err := strconv.ParseInt(updateFlags.Arg(0), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid feed id %q", updateFlags.Arg(0))
		}
		update := miniflux.UpdateFeedInput{}
		if *feedURL != "" {
			update.FeedURL = feedURL
		}
		if *siteURL != "" {
			update.SiteURL = siteURL
		}
		if *title != "" {
			update.Title = title
		}
		if *categoryID > 0 {
			update.CategoryID = categoryID
		}
		if *username != "" {
			update.Username = username
		}
		if *password != "" {
			update.Password = password
		}
		if *userAgent != "" {
			update.UserAgent = userAgent
		}
		if update.FeedURL == nil && update.SiteURL == nil && update.Title == nil && update.CategoryID == nil && update.Username == nil && update.Password == nil && update.UserAgent == nil {
			_, _ = fmt.Fprintln(stderr, "feed update requires at least one field flag to update")
			return ErrUsage
		}
		feed, err := client.UpdateFeed(feedID, update)
		if err != nil {
			return err
		}
		return printJSON(stdout, feed)
	case "delete":
		if len(args) != 2 {
			_, _ = fmt.Fprintln(stderr, "usage: miniflux-cli feed delete <feed-id>")
			return ErrUsage
		}
		feedID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid feed id %q", args[1])
		}
		if err := client.DeleteFeed(feedID); err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "{\"deleted\": %d}\n", feedID)
		return err
	case "refresh":
		if len(args) != 2 {
			_, _ = fmt.Fprintln(stderr, "usage: miniflux-cli feed refresh <feed-id>")
			return ErrUsage
		}
		feedID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid feed id %q", args[1])
		}
		if err := client.RefreshFeed(feedID); err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "{\"refreshed\": %d}\n", feedID)
		return err
	default:
		return fmt.Errorf("unknown feed subcommand %q", args[0])
	}
}
