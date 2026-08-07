package api

import (
	"io"
	"iter"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestFetchPRComments(t *testing.T) {
	transport := localRoundTripper{
		handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/graphql" {
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
	}
	api := API{
		defaultTransport: transport,
	}

	res, err := api.FetchPR("https://github.com/some/repo/pull/12345")
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

func TestFetchPRDiff(t *testing.T) {
	transport := localRoundTripper{
		handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/repos/some/repo/pulls/12345" {
				t.Fatalf("Incorrect path %s", r.URL.Path)
			}

			d, err := os.ReadFile("./testdata/prDiff.diff")
			if err != nil {
				t.Errorf("failed reading mock pr diff %v", err)
			}
			val := string(d)
			mustWrite(t, w, val)
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/vnd.github.v3.diff")
		}),
	}

	api := API{
		defaultTransport: transport,
	}

	res, err := api.FetchPRDiff("https://github.com/some/repo/pull/12345")
	if err != nil {
		t.Fatal(err)
	}

	next, stop := iter.Pull(strings.Lines(res))
	defer stop()
	line, _ := next()
	if line != "diff --git a/.github/actions/setup/action.yml b/.github/actions/setup/action.yml\n" {
		t.Fatalf(
			`expected a PR diff that starts with "diff --git a/.github/actions/setup/action.yml b/.github/actions/setup/action.yml" but got "%s"`,
			line,
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
