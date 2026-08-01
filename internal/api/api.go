package api

import (
	"net/http"
	"os"

	gh "github.com/cli/go-gh/v2/pkg/api"
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
	opts := gh.ClientOptions{}
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
	opts := gh.ClientOptions{}
	if level == "debug" {
		logger := NewHTTPLogger(0)
		opts.Log = &logger
		opts.LogVerboseHTTP = true
		opts.LogColorize = true
	}

	a.httpClient, err = gh.NewHTTPClient(opts)
	return a.httpClient, err
}
