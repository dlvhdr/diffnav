package api

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"charm.land/log/v2"
	gh "github.com/cli/go-gh/v2/pkg/api"
	"github.com/shurcooL/githubv4"
)

type API struct {
	url        string
	gqlClient  *gh.GraphQLClient
	httpClient *http.Client
}

const (
	defaultAPIURL    = "https://api.github.com"
	defaultServerURL = "https://github.com"
)

func New() API {
	apiURL := os.Getenv("GITHUB_API_URL")
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	a := API{}

	// initialize singletons
	a.getHTTPClient()
	a.getGraphQLClient()

	a.url = apiURL

	return a
}

func (a *API) SetClient(c *gh.GraphQLClient) {
	a.gqlClient = c
}

func (a *API) getGraphQLClient() (*gh.GraphQLClient, error) {
	var err error
	if a.gqlClient != nil {
		return a.gqlClient, nil
	}

	level := os.Getenv("LOG_LEVEL")
	opts := gh.ClientOptions{Host: a.url}
	if level == "debug" {
		logger := NewHTTPLogger(0)
		opts.Log = &logger
		opts.LogVerboseHTTP = true
		opts.LogColorize = true
	}
	a.gqlClient, err = gh.NewGraphQLClient(opts)
	return a.gqlClient, err
}

func (a *API) getHTTPClient() (*http.Client, error) {
	var err error
	if a.httpClient != nil {
		return a.httpClient, nil
	}
	level := os.Getenv("LOG_LEVEL")
	opts := gh.ClientOptions{Host: a.url}
	if level == "debug" {
		logger := NewHTTPLogger(0)
		opts.Log = &logger
		opts.LogVerboseHTTP = true
		opts.LogColorize = true
	}

	a.httpClient, err = gh.NewHTTPClient(opts)
	return a.httpClient, err
}

const (
	DiffSideLeft  = "LEFT"
	DiffSideRight = "RIGHT"
)

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

func (a *API) FetchPR(repo string, prNumber string) (PRQuery, error) {
	var err error
	var res PRQuery
	c, err := a.getGraphQLClient()
	if err != nil {
		return res, err
	}

	prURL, err := url.Parse(fmt.Sprintf("https://github.com/%s/pull/%s", repo, prNumber))
	if err != nil {
		return res, err
	}
	variables := map[string]any{
		"url": githubv4.URI{URL: prURL},
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
