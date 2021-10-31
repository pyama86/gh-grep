package gh

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v33/github"
)

type Gh struct {
	client *github.Client
}

func New() (*Gh, error) {

	var token, v3ep string
	if EnvsNotEmpty("CI", "GITHUB_ACTION", "GITHUB_API_URL", "GITHUB_TOKEN") {
		// GitHub Actions
		token = os.Getenv("GITHUB_TOKEN")
		v3ep = os.Getenv("GITHUB_API_URL")
	} else if EnvsNotEmpty("GH_HOST", "GH_ENTERPRISE_TOKEN") || EnvsNotEmpty("GH_HOST", "GITHUB_ENTERPRISE_TOKEN") {
		// GitHub Enterprise Server
		token = os.Getenv("GH_ENTERPRISE_TOKEN")
		if token == "" {
			token = os.Getenv("GITHUB_ENTERPRISE_TOKEN")
		}
		v3ep = fmt.Sprintf("https://%s/api/v3", os.Getenv("GH_HOST"))
	} else {
		// GitHub.com
		token = os.Getenv("GH_TOKEN")
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}
	}

	if token == "" {
		return nil, fmt.Errorf("env %s is not set", "GITHUB_TOKEN")
	}

	v3c := github.NewClient(httpClient(token))
	if v3ep != "" {
		baseEndpoint, err := url.Parse(v3ep)
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(baseEndpoint.Path, "/") {
			baseEndpoint.Path += "/"
		}
		v3c.BaseURL = baseEndpoint
	}

	return &Gh{
		client: v3c,
	}, nil
}

func (g *Gh) Client() *github.Client {
	return g.client
}

func (g *Gh) Repositories(ctx context.Context, owner string) ([]string, error) {
	repos := []string{}

	page := 1
	for {
		rs, res, err := g.client.Repositories.List(ctx, owner, &github.RepositoryListOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		})
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			repos = append(repos, *r.Name)
		}
		if res.NextPage == 0 {
			break
		}
		page = res.NextPage
	}

	return repos, nil
}

func EnvsNotEmpty(keys ...string) bool {
	for _, k := range keys {
		if os.Getenv(k) == "" {
			return false
		}
	}
	return true
}

type roundTripper struct {
	transport   *http.Transport
	accessToken string
}

func (rt roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", fmt.Sprintf("token %s", rt.accessToken))
	return rt.transport.RoundTrip(r)
}

func httpClient(token string) *http.Client {
	t := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	rt := roundTripper{
		transport:   t,
		accessToken: token,
	}
	return &http.Client{
		Timeout:   time.Second * 30,
		Transport: rt,
	}
}
