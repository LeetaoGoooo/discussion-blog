package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// GitHubFetcher fetches data from GitHub Discussions
type GitHubFetcher struct {
	client *githubv4.Client
	owner  string
	repo   string
}

// NewGitHubFetcher creates a new GitHubFetcher
func NewGitHubFetcher(owner, repo, token string) *GitHubFetcher {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	return &GitHubFetcher{
		client: client,
		owner:  owner,
		repo:   repo,
	}
}

// Discussion represents a GitHub Discussion
type Discussion struct {
	ID       string
	Number   int
	Title    string
	Body     string
	Author   string
	Category struct {
		ID   string
		Name string
	}
	Labels []struct {
		Name string
	}
	CreatedAt time.Time
	URL       string
}

// FetchDiscussions fetches discussions from GitHub
func (g *GitHubFetcher) FetchDiscussions() ([]Discussion, error) {
	var query struct {
		Repository struct {
			Discussions struct {
				Nodes []struct {
					ID        string
					Number    int
					Title     string
					Body      string
					Author    struct {
						Login string
					}
					Category struct {
						ID   string
						Name string
					}
					Labels struct {
						Nodes []struct {
							Name string
						}
					} `graphql:"labels(first: 10)"`
					CreatedAt time.Time
					URL       string
				}
				PageInfo struct {
					EndCursor   string
					HasNextPage bool
				}
			} `graphql:"discussions(first: 100, after: $cursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]interface{}{
		"owner":  githubv4.String(g.owner),
		"name":   githubv4.String(g.repo),
		"cursor": (*githubv4.String)(nil),
	}

	var discussions []Discussion

	for {
		err := g.client.Query(context.Background(), &query, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch discussions: %w", err)
		}

		for _, node := range query.Repository.Discussions.Nodes {
			labels := make([]struct{ Name string }, len(node.Labels.Nodes))
			for i, label := range node.Labels.Nodes {
				labels[i] = struct{ Name string }{Name: label.Name}
			}

			discussions = append(discussions, Discussion{
				ID:    node.ID,
				Number: node.Number,
				Title: node.Title,
				Body:  node.Body,
				Author: node.Author.Login,
				Category: struct {
					ID   string
					Name string
				}{
					ID:   node.Category.ID,
					Name: node.Category.Name,
				},
				Labels:    labels,
				CreatedAt: node.CreatedAt,
				URL:       node.URL,
			})
		}

		if !query.Repository.Discussions.PageInfo.HasNextPage {
			break
		}

		variables["cursor"] = githubv4.String(query.Repository.Discussions.PageInfo.EndCursor)
	}

	return discussions, nil
}