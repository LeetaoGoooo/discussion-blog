package generator

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/russross/blackfriday/v2"
	"pure/internal/fetcher"
	"pure/internal/utils"
)

// Config represents the site configuration
type Config struct {
	Site struct {
		Title       string
		URL         string
		Description string
		Author      string
		Email       string
		AboutID     int
		Giscus      struct {
			RepoID     string
			Category   string
			CategoryID string
		}
		Favicon string
	}
	Github struct {
		Owner string
		Repo  string
	}
}

// SiteGenerator generates static site files
type SiteGenerator struct {
	config      Config
	templateDir string
	outputDir   string
	templates   *template.Template
}

// NewSiteGenerator creates a new SiteGenerator
func NewSiteGenerator(config Config, templateDir, outputDir string) (*SiteGenerator, error) {
	// Define custom template functions
	funcMap := template.FuncMap{
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"markdown": func(s string) template.HTML {
			// Clean up line endings
			s = strings.ReplaceAll(s, "\r\n", "\n")
			s = strings.ReplaceAll(s, "\r", "\n")
			
			// Convert markdown to HTML
			html := blackfriday.Run([]byte(s))
			return template.HTML(html)
		},
		"trimBraces": func(s string) string {
			s = strings.TrimPrefix(s, "{")
			s = strings.TrimSuffix(s, "}")
			return s
		},
	}
	
	// Parse all templates from the template directory with custom functions
	templates, err := template.New("").Funcs(funcMap).ParseGlob(templateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	
	return &SiteGenerator{
		config:      config,
		templateDir: templateDir,
		outputDir:   outputDir,
		templates:   templates,
	}, nil
}

// Generate generates the static site
func (g *SiteGenerator) Generate(discussions []fetcher.Discussion) error {
	// Create output directory
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate index page
	if err := g.generateIndexPage(discussions); err != nil {
		return fmt.Errorf("failed to generate index page: %w", err)
	}

	// Generate individual post pages
	if err := g.generatePostPages(discussions); err != nil {
		return fmt.Errorf("failed to generate post pages: %w", err)
	}

	// Generate about page if about_id is configured
	if err := g.generateAboutPage(discussions); err != nil {
		return fmt.Errorf("failed to generate about page: %w", err)
	}

	// Generate tag page
	if err := g.generateTagPage(discussions); err != nil {
		return fmt.Errorf("failed to generate tag page: %w", err)
	}

	// Generate RSS feed
	if err := g.generateRSSFeed(discussions); err != nil {
		return fmt.Errorf("failed to generate RSS feed: %w", err)
	}

	// Generate search index
	if err := g.generateSearchIndex(discussions); err != nil {
		return fmt.Errorf("failed to generate search index: %w", err)
	}

	// Copy static assets
	if err := g.copyStaticAssets(); err != nil {
		return fmt.Errorf("failed to copy static assets: %w", err)
	}

	return nil
}

func (g *SiteGenerator) generateIndexPage(discussions []fetcher.Discussion) error {
	// Filter out about page if about_id is configured
	var filteredDiscussions []fetcher.Discussion
	for _, discussion := range discussions {
		// Skip the discussion if it matches the about_id
		if g.config.Site.AboutID > 0 && discussion.Number == g.config.Site.AboutID {
			continue
		}
		filteredDiscussions = append(filteredDiscussions, discussion)
	}
	
	// Use filtered discussions for index page generation
	discussions = filteredDiscussions
	
	// Define posts per page
	postsPerPage := 10
	
	// Calculate total pages
	totalPosts := len(discussions)
	totalPages := (totalPosts + postsPerPage - 1) / postsPerPage
	
	// Generate pages
	for page := 1; page <= totalPages; page++ {
		// Calculate start and end indices
		start := (page - 1) * postsPerPage
		end := start + postsPerPage
		if end > totalPosts {
			end = totalPosts
		}
		
		// Get posts for this page
		pageDiscussions := discussions[start:end]
		
		// Create page directory
		var pageDir string
		if page == 1 {
			// First page goes to root directory (index page)
			pageDir = g.outputDir
		} else {
			// Other pages go to /page/{page}/
			pageDir = filepath.Join(g.outputDir, "page", fmt.Sprintf("%d", page))
		}
		
		if err := os.MkdirAll(pageDir, 0755); err != nil {
			return fmt.Errorf("failed to create page directory: %w", err)
		}

		// Create index.html
		indexPath := filepath.Join(pageDir, "index.html")
		file, err := os.Create(indexPath)
		if err != nil {
			return fmt.Errorf("failed to create index.html: %w", err)
		}
		defer file.Close()

		// Prepare data for template
	data := struct {
		Site struct {
			Site struct {
				Title       string
				URL         string
				Description string
				Author      string
				Email       string
				AboutID     int
				Giscus      struct {
					RepoID     string
					Category   string
					CategoryID string
				}
				Favicon string
			}
		}
		Discussions []fetcher.Discussion
		Pagination  struct {
			CurrentPage int
			TotalPages  int
			HasPrev     bool
			HasNext     bool
			PrevPage    int
			NextPage    int
		}
	}{
		Site: struct {
			Site struct {
				Title       string
				URL         string
				Description string
				Author      string
				Email       string
				AboutID     int
				Giscus      struct {
					RepoID     string
					Category   string
					CategoryID string
				}
				Favicon string
			}
		}{
			Site: g.config.Site,
		},
		Discussions: pageDiscussions,
		Pagination: struct {
			CurrentPage int
			TotalPages  int
			HasPrev     bool
			HasNext     bool
			PrevPage    int
			NextPage    int
		}{
			CurrentPage: page,
			TotalPages:  totalPages,
			HasPrev:     page > 1,
			HasNext:     page < totalPages,
			PrevPage:    page - 1,
			NextPage:    page + 1,
		},
	}

		// Execute the index template
		if err := g.templates.ExecuteTemplate(file, "index.html", data); err != nil {
			return fmt.Errorf("failed to execute index template: %w", err)
		}
	}

	return nil
}

func (g *SiteGenerator) generatePostPages(discussions []fetcher.Discussion) error {
	for _, discussion := range discussions {
		// Create post directory
		postDir := filepath.Join(g.outputDir, "post", fmt.Sprintf("%d", discussion.Number))
		if err := os.MkdirAll(postDir, 0755); err != nil {
			return fmt.Errorf("failed to create post directory: %w", err)
		}

		// Create index.html
		postPath := filepath.Join(postDir, "index.html")
		file, err := os.Create(postPath)
		if err != nil {
			return fmt.Errorf("failed to create post index.html: %w", err)
		}
		defer file.Close()

		// Prepare data for template
		data := struct {
			Site struct {
				Site struct {
					Title       string
					URL         string
					Description string
					Author      string
					Email       string
					AboutID     int
					Giscus      struct {
						RepoID     string
						Category   string
						CategoryID string
					}
					Favicon string
				}
				Github struct {
					Owner string
					Repo  string
				}
			}
			Discussion fetcher.Discussion
		}{
			Site: struct {
				Site struct {
					Title       string
					URL         string
					Description string
					Author      string
					Email       string
					AboutID     int
					Giscus      struct {
						RepoID     string
						Category   string
						CategoryID string
					}
					Favicon string
				}
				Github struct {
					Owner string
					Repo  string
				}
			}{
				Site: g.config.Site,
				Github: struct {
					Owner string
					Repo  string
				}{
					Owner: g.config.Github.Owner,
					Repo:  g.config.Github.Repo,
				},
			},
			Discussion: discussion,
		}

		// Execute the post template
		if err := g.templates.ExecuteTemplate(file, "post.html", data); err != nil {
			return fmt.Errorf("failed to execute post template: %w", err)
		}
	}

	return nil
}

func (g *SiteGenerator) generateTagPage(discussions []fetcher.Discussion) error {
	// Collect all unique tags
	tagMap := make(map[string]int)
	for _, discussion := range discussions {
		for _, label := range discussion.Labels {
			tagMap[label.Name]++
		}
	}

	// Create tags directory
	tagsDir := filepath.Join(g.outputDir, "tags")
	if err := os.MkdirAll(tagsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tags directory: %w", err)
	}

	// Create index.html
	tagsPath := filepath.Join(tagsDir, "index.html")
	file, err := os.Create(tagsPath)
	if err != nil {
		return fmt.Errorf("failed to create tags index.html: %w", err)
	}
	defer file.Close()

	// Prepare data for template
	data := struct {
		Site struct {
			Site struct {
				Title       string
				URL         string
				Description string
				Author      string
				Email       string
				AboutID     int
				Giscus      struct {
					RepoID     string
					Category   string
					CategoryID string
				}
				Favicon string
			}
		}
		Tags map[string]int
	}{
		Site: struct {
			Site struct {
				Title       string
				URL         string
				Description string
				Author      string
				Email       string
				AboutID     int
				Giscus      struct {
					RepoID     string
					Category   string
					CategoryID string
				}
				Favicon string
			}
		}{
			Site: g.config.Site,
		},
		Tags: tagMap,
	}

	// Execute the tags template
	if err := g.templates.ExecuteTemplate(file, "tags.html", data); err != nil {
		return fmt.Errorf("failed to execute tags template: %w", err)
	}

	// Generate individual tag pages
	for tag := range tagMap {
		if err := g.generateTagPageForTag(tag, discussions); err != nil {
			return fmt.Errorf("failed to generate tag page for %s: %w", tag, err)
		}
	}

	return nil
}

func (g *SiteGenerator) generateTagPageForTag(tag string, discussions []fetcher.Discussion) error {
	// Create tag directory
	tagDir := filepath.Join(g.outputDir, "tags", tag)
	if err := os.MkdirAll(tagDir, 0755); err != nil {
		return fmt.Errorf("failed to create tag directory: %w", err)
	}

	// Filter discussions by tag
	var taggedDiscussions []fetcher.Discussion
	for _, discussion := range discussions {
		for _, label := range discussion.Labels {
			if label.Name == tag {
				taggedDiscussions = append(taggedDiscussions, discussion)
				break
			}
		}
	}

	// Create index.html
	tagPath := filepath.Join(tagDir, "index.html")
	file, err := os.Create(tagPath)
	if err != nil {
		return fmt.Errorf("failed to create tag index.html: %w", err)
	}
	defer file.Close()

	// Prepare data for template
	data := struct {
		Site struct {
			Site struct {
				Title       string
				URL         string
				Description string
				Author      string
				Email       string
				AboutID     int
				Giscus      struct {
					RepoID     string
					Category   string
					CategoryID string
				}
				Favicon string
			}
		}
		Tag         string
		Discussions []fetcher.Discussion
	}{
		Site: struct {
			Site struct {
				Title       string
				URL         string
				Description string
				Author      string
				Email       string
				AboutID     int
				Giscus      struct {
					RepoID     string
					Category   string
					CategoryID string
				}
				Favicon string
			}
		}{
			Site: g.config.Site,
		},
		Tag:         tag,
		Discussions: taggedDiscussions,
	}

	// Execute the tag template
	if err := g.templates.ExecuteTemplate(file, "tag.html", data); err != nil {
		return fmt.Errorf("failed to execute tag template: %w", err)
	}

	return nil
}

func (g *SiteGenerator) generateRSSFeed(discussions []fetcher.Discussion) error {
	// Create RSS feed
	rssPath := filepath.Join(g.outputDir, "atom.xml")
	file, err := os.Create(rssPath)
	if err != nil {
		return fmt.Errorf("failed to create atom.xml: %w", err)
	}
	defer file.Close()

	// Atom feed template
	rssTemplate := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
	<title>{{.Site.Title}}</title>
	<subtitle>{{.Site.Description}}</subtitle>
	<link href="{{.Site.URL}}"/>
	<link href="{{.Site.URL}}/atom.xml" rel="self"/>
	<updated>{{.Updated}}</updated>
	<id>{{.Site.URL}}</id>
	<author>
		<name>{{.Site.Author}}</name>
		<email>{{.Site.Email}}</email>
	</author>
	{{range .Discussions}}
	<entry>
		<title>{{.Title}}</title>
		<link href="{{.URL}}"/>
		<link href="{{.URL}}" rel="alternate"/>
		<id>{{.URL}}</id>
		<published>{{.CreatedAt.Format "2006-01-02T15:04:05Z07:00"}}</published>
		<updated>{{.CreatedAt.Format "2006-01-02T15:04:05Z07:00"}}</updated>
		<summary type="html">{{.Body | escapeHTML}}</summary>
	</entry>
	{{end}}
</feed>
`

	tmpl, err := template.New("atom").Funcs(template.FuncMap{
		"escapeHTML": func(s string) template.HTML {
			// Convert markdown to HTML and escape HTML entities
			html := blackfriday.Run([]byte(s))
			// Remove HTML tags for summary
			re := regexp.MustCompile("<[^>]*>")
			summary := re.ReplaceAllString(string(html), "")
			// Truncate summary to 200 characters
			if len(summary) > 200 {
				summary = summary[:200] + "..."
			}
			return template.HTML(template.HTMLEscapeString(summary))
		},
	}).Parse(rssTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse Atom template: %w", err)
	}

	// Get the most recent updated time
	var updated time.Time
	for _, discussion := range discussions {
		if discussion.CreatedAt.After(updated) {
			updated = discussion.CreatedAt
		}
	}

	data := struct {
		Site struct {
			Title       string
			Description string
			URL         string
			Author      string
			Email       string
		}
		Updated     string
		Discussions []fetcher.Discussion
	}{
		Site: struct {
			Title       string
			Description string
			URL         string
			Author      string
			Email       string
		}{
			Title:       g.config.Site.Title,
			Description: g.config.Site.Description,
			URL:         g.config.Site.URL,
			Author:      g.config.Site.Author,
			Email:       g.config.Site.Email,
		},
		Updated:     updated.Format("2006-01-02T15:04:05Z07:00"),
		Discussions: discussions,
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute Atom template: %w", err)
	}

	return nil
}

func (g *SiteGenerator) generateSearchIndex(discussions []fetcher.Discussion) error {
	// Create search index
	searchIndexPath := filepath.Join(g.outputDir, "search-index.json")
	file, err := os.Create(searchIndexPath)
	if err != nil {
		return fmt.Errorf("failed to create search-index.json: %w", err)
	}
	defer file.Close()

	// Convert discussions to search index format
	var searchIndex []map[string]interface{}
	for _, discussion := range discussions {
		labels := make([]string, len(discussion.Labels))
		for i, label := range discussion.Labels {
			labels[i] = label.Name
		}

		searchIndex = append(searchIndex, map[string]interface{}{
			"id":       discussion.Number,
			"title":    discussion.Title,
			"content":  utils.PreviewContent(discussion.Body),
			"category": discussion.Category.Name,
			"labels":   labels,
			"date":     discussion.CreatedAt.Format("2006-01-02"),
		})
	}

	// Write JSON to file
	jsonData, err := utils.ToJSON(searchIndex)
	if err != nil {
		return fmt.Errorf("failed to convert search index to JSON: %w", err)
	}

	if _, err := file.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write search index: %w", err)
	}

	return nil
}

func (g *SiteGenerator) copyStaticAssets() error {
	// Use absolute path to public directory
	publicDir := "./public"
	
	// Check if public directory exists
	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		// If not, try alternative path
		publicDir = "../public"
		if _, err := os.Stat(publicDir); os.IsNotExist(err) {
			return fmt.Errorf("public directory not found")
		}
	}
	
	// Copy all files from public directory to output directory
	return filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Calculate relative path
		relPath, err := filepath.Rel(publicDir, path)
		if err != nil {
			return err
		}
		
		// Create destination path
		destPath := filepath.Join(g.outputDir, relPath)
		
		// Create destination directory if it doesn't exist
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}
		
		// Read source file
		srcData, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		
		// Write to destination
		if err := ioutil.WriteFile(destPath, srcData, 0644); err != nil {
			return err
		}
		
		return nil
	})
}

func (g *SiteGenerator) generateAboutPage(discussions []fetcher.Discussion) error {
	// Check if about_id is configured
	if g.config.Site.AboutID <= 0 {
		return nil // No about page configured
	}

	// Find the discussion with the specified ID
	var aboutDiscussion *fetcher.Discussion
	for _, discussion := range discussions {
		if discussion.Number == g.config.Site.AboutID {
			aboutDiscussion = &discussion
			break
		}
	}

	// If not found, return without error
	if aboutDiscussion == nil {
		return nil
	}

	// Create about directory
	aboutDir := filepath.Join(g.outputDir, "about")
	if err := os.MkdirAll(aboutDir, 0755); err != nil {
		return fmt.Errorf("failed to create about directory: %w", err)
	}

	// Create index.html
	aboutPath := filepath.Join(aboutDir, "index.html")
	file, err := os.Create(aboutPath)
	if err != nil {
		return fmt.Errorf("failed to create about index.html: %w", err)
	}
	defer file.Close()

	// Prepare data for template
	data := struct {
		Site struct {
			Site struct {
				Title       string
				URL         string
				Description string
				Author      string
				Email       string
				AboutID     int
				Giscus      struct {
					RepoID     string
					Category   string
					CategoryID string
				}
				Favicon string
			}
			Github struct {
				Owner string
				Repo  string
			}
		}
		Discussion fetcher.Discussion
	}{
		Site: struct {
			Site struct {
				Title       string
				URL         string
				Description string
				Author      string
				Email       string
				AboutID     int
				Giscus      struct {
					RepoID     string
					Category   string
					CategoryID string
				}
				Favicon string
			}
			Github struct {
				Owner string
				Repo  string
			}
		}{
			Site: g.config.Site,
			Github: struct {
				Owner string
				Repo  string
			}{
				Owner: g.config.Github.Owner,
				Repo:  g.config.Github.Repo,
			},
		},
		Discussion: *aboutDiscussion,
	}

	// Execute the post template for about page
	if err := g.templates.ExecuteTemplate(file, "post.html", data); err != nil {
		return fmt.Errorf("failed to execute post template for about page: %w", err)
	}

	return nil
}