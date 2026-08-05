package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	gh "github.com/cli/go-gh/v2/pkg/api"
)

func TestFetchPRComments(t *testing.T) {
	gqlClient, err := gh.NewGraphQLClient(gh.ClientOptions{
		Transport: localRoundTripper{
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/graphql" {
					t.Fatalf("Incorrect path %s", r.URL.Path)
				}

				body := mustRead(t, r.Body)
				switch {
				case strings.Contains(body, "query FetchPRComments"):
					t.Log("matched query FetchPRComments")
					d, err := os.ReadFile("./testdata/prComments.json")
					if err != nil {
						t.Errorf("failed reading mock data file %v", err)
					}
					mustWrite(t, w, string(d))
				default:
					t.Log("unexpected url", r.URL)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
			}),
		},
		Host:      "localhost:3000",
		AuthToken: "fake-token",
	})
	api := API{
		gqlClient: gqlClient,
	}

	res, err := api.FetchPR("some/repo", "12345")
	if err != nil {
		t.Fatal(err)
	}

	if res.Resource.PullRequest.Title != "feat(terminal): replace `libvterm` with `libghostty-vt`" {
		t.Fatalf(
			`expected a PR with title "%s" but got "%s"`,
			"feat(terminal): replace `libvterm` with `libghostty-vt`",
			res.Resource.PullRequest.Title,
		)
	}

	if len(res.Resource.PullRequest.ReviewThreads.Nodes) != 10 {
		t.Fatalf(
			`expected a PR with 10 review threads but got %d`,
			len(res.Resource.PullRequest.ReviewThreads.Nodes),
		)
	}
}

func mustRead(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func mustWrite(t *testing.T, w io.Writer, s string) {
	t.Helper()
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}

// localRoundTripper is an http.RoundTripper that executes HTTP transactions
// by using handler directly, instead of going over an HTTP connection.
// This is used because the github graphql client expects an authentication token
type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}
