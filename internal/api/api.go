package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"charm.land/log/v2"
	gh "github.com/cli/go-gh/v2/pkg/api"
	"github.com/shurcooL/githubv4"
)

type API struct {
	Host             string
	defaultTransport http.RoundTripper
}

const (
	defaultAPIURL    = "https://api.github.com"
	defaultServerURL = "https://github.com"
)

var prURLPattern = regexp.MustCompile(
	`^/(?P<owner>[^/]+)/(?P<repo>[^/]+)/pull/(?P<number>\d+)`,
)

func New() API {
	apiURL := os.Getenv("GITHUB_API_URL")
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	a := API{
		Host: apiURL,
	}

	return a
}

func (a *API) setDefaultTransport(transport *http.Transport) {
	a.defaultTransport = transport
}

func (a *API) getGraphQLClient(opts gh.ClientOptions) (*gh.GraphQLClient, error) {
	if a.defaultTransport != nil {
		opts.Transport = a.defaultTransport
	}
	opts.Host = a.Host

	level := os.Getenv("LOG_LEVEL")
	if level == "debug" {
		logger := NewHTTPLogger(0)
		opts.Log = &logger
		opts.LogVerboseHTTP = true
		opts.LogColorize = true
	}
	return gh.NewGraphQLClient(opts)
}

func (a *API) getRESTClient(opts gh.ClientOptions) (*gh.RESTClient, error) {
	if a.defaultTransport != nil {
		opts.Transport = a.defaultTransport
	}

	level := os.Getenv("LOG_LEVEL")
	if level == "debug" {
		logger := NewHTTPLogger(0)
		opts.Log = &logger
		opts.LogVerboseHTTP = true
		opts.LogColorize = true
	}

	return gh.NewRESTClient(opts)
}

func (a *API) getClient(opts gh.ClientOptions) (*http.Client, error) {
	if a.defaultTransport != nil {
		opts.Transport = a.defaultTransport
	}

	level := os.Getenv("LOG_LEVEL")
	if level == "debug" {
		logger := NewHTTPLogger(0)
		opts.Log = &logger
		opts.LogVerboseHTTP = true
		opts.LogColorize = true
	}

	return gh.NewHTTPClient(opts)
}

const (
	DiffSideLeft  = "LEFT"
	DiffSideRight = "RIGHT"
)

func (a *API) FetchPR(prURL string) (PRQuery, error) {
	var err error
	var res PRQuery
	c, err := a.getGraphQLClient(gh.ClientOptions{})
	if err != nil {
		return res, err
	}

	parsedPRURL, err := url.Parse(prURL)
	if err != nil {
		return res, err
	}
	variables := map[string]any{
		"url": githubv4.URI{URL: parsedPRURL},
	}

	startTime := time.Now()
	err = c.Query("FetchPRComments", &res, variables)
	if err != nil {
		log.Error("error fetching PR", "err", err)
		return res, err
	}

	log.Debug("FetchPR request completed", "duration", time.Since(startTime))
	return res, nil
}

type diffResp struct{ Body string }

func (a *API) FetchPRDiff(prURL string) (string, error) {
	c, err := a.getClient(gh.ClientOptions{Headers: map[string]string{
		"Accept": "application/vnd.github.v3.diff",
	}})
	if err != nil {
		return "", err
	}

	parsedPRURL, err := url.Parse(prURL)
	if err != nil {
		return "", err
	}

	// PRURL to diff url
	// https://github.com/neovim/neovim/pull/39773 ->
	// https://api.github.com/repos/neovim/neovim/pulls/7
	match := prURLPattern.FindStringSubmatch(parsedPRURL.Path)
	if match == nil {
		return "", fmt.Errorf("failed parsing pr url %s", prURL)
	}

	repo := match[prURLPattern.SubexpIndex("owner")] + "/" +
		match[prURLPattern.SubexpIndex("repo")]
	prNumber := match[prURLPattern.SubexpIndex("number")]

	u := fmt.Sprintf("%s/repos/%s/pulls/%s", a.Host, repo, prNumber)
	log.Debug("fetching", "url", u)
	resp, err := c.Get(u)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed fetching pr diff: %s", resp.Status)
	}

	return string(b), nil
}

type ReviewThread struct {
	Id           string
	IsResolved   bool
	IsOutdated   bool
	Path         string
	Line         int
	OriginalLine int
	DiffSide     string
	Comments     struct {
		Nodes []ReviewThreadComment
	} `graphql:"comments(first: 100)"`
}

type ReviewThreadComment struct {
	Id        string
	Body      string
	Author    struct{ Login string }
	CreatedAt time.Time
	Url       string
	ReplyTo   struct{ Id string }
}

type PR struct {
	Title      string
	Number     int
	Url        string
	Repository struct {
		NameWithOwner string
	}
	Merged        bool
	IsDraft       bool
	Closed        bool
	HeadRefName   string
	ReviewThreads struct {
		Nodes []ReviewThread
	} `graphql:"reviewThreads(first: 10)"`
}

type PRQuery struct {
	Resource struct {
		PullRequest PR `graphql:"... on PullRequest"`
	} `graphql:"resource(url: $url)"`
}
