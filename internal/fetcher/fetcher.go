package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// FetchDiscussions fetches discussions from GitHub
func FetchDiscussions(token, owner, repo string) ([]Discussion, error) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

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
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(repo),
		"cursor": (*githubv4.String)(nil),
	}

	var discussions []Discussion

	for {
		err := client.Query(context.Background(), &query, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch discussions: %w", err)
		}

		for _, node := range query.Repository.Discussions.Nodes {
			labels := make([]struct{ Name string }, len(node.Labels.Nodes))
			for i, label := range node.Labels.Nodes {
				// Remove braces from label name if present
				name := label.Name
				if len(name) >= 2 && name[0] == '{' && name[len(name)-1] == '}' {
					name = name[1 : len(name)-1]
				}
				labels[i] = struct{ Name string }{Name: name}
				
				// Debugging: Print the first article's labels
				if node.Number == 2 && i < 2 {
					fmt.Printf("Article %d, Label %d: Original=%s, Processed=%s\n", node.Number, i, label.Name, name)
				}
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