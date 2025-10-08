package generator

import (
	"bytes"
	"fmt"
	"html/template"
	texttemplate "text/template"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"pure/internal/fetcher"
	"pure/internal/utils"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/russross/blackfriday/v2"
)

// Site represents the site-specific configuration.
type Site struct {
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
	Favicon  string
	Language string
}

// Github represents the GitHub-specific configuration.
type Github struct {
	Owner string
	Repo  string
}

// Config represents the site configuration
type Config struct {
	Site   Site
	Github Github
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
		"default": func(defaultValue, value interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
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

			// Convert markdown to HTML with Chroma
			renderer := &ChromaRenderer{HTML: blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
				Flags: blackfriday.UseXHTML,
			})}
			extensions := blackfriday.CommonExtensions | blackfriday.AutoHeadingIDs | blackfriday.NoEmptyLineBeforeBlock
			html := blackfriday.Run([]byte(s), blackfriday.WithRenderer(renderer), blackfriday.WithExtensions(extensions))
			return template.HTML(html)
		},
		"trimBraces": func(s string) string {
			s = strings.TrimPrefix(s, "{")
			s = strings.TrimSuffix(s, "}")
			return s
		},
		"truncateHTML": func(s string, length int) string {
			// Convert markdown to HTML first
			htmlContent := blackfriday.Run([]byte(s))
			// Remove HTML tags
			re := regexp.MustCompile("<[^>]*>")
			plainText := re.ReplaceAllString(string(htmlContent), "")
			// Truncate to specified length
			if len(plainText) > length {
				plainText = plainText[:length] + "..."
			}
			return plainText
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

// ChromaRenderer is a custom Blackfriday renderer that uses Chroma for syntax highlighting
type ChromaRenderer struct {
	HTML blackfriday.Renderer
}

func (r *ChromaRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	if node.Type == blackfriday.CodeBlock {
		var lang string
		if node.CodeBlockData.Info != nil {
			info := string(node.CodeBlockData.Info)
			// Take only the first token (strip params like "swift title=...")
			if fields := strings.Fields(info); len(fields) > 0 {
				lang = fields[0]
			}
			// Normalize common prefixes
			lang = strings.TrimPrefix(lang, "language-")
		}

		// Prefer exact match first
		lexer := lexers.Get(lang)
		// If no language provided or not found, try to auto-detect
		if lexer == nil || lang == "" {
			if analysed := lexers.Analyse(string(node.Literal)); analysed != nil {
				lexer = analysed
			}
		}
		if lexer == nil {
			lexer = lexers.Fallback
		}

		style := styles.Get("github")
		if style == nil {
			style = styles.Fallback
		}

		formatter := html.New(html.WithClasses(true))

		iterator, err := lexer.Tokenise(nil, string(node.Literal))
		if err != nil {
			return r.HTML.RenderNode(w, node, entering)
		}

		buf := new(bytes.Buffer)
		if err := formatter.Format(buf, style, iterator); err != nil {
			return r.HTML.RenderNode(w, node, entering)
		}
		w.Write(buf.Bytes())

		return blackfriday.GoToNext
	}

	return r.HTML.RenderNode(w, node, entering)
}

func (r *ChromaRenderer) RenderHeader(w io.Writer, ast *blackfriday.Node) {
	r.HTML.RenderHeader(w, ast)
}

func (r *ChromaRenderer) RenderFooter(w io.Writer, ast *blackfriday.Node) {
	r.HTML.RenderFooter(w, ast)
}

// Generate generates the static site
func (g *SiteGenerator) Generate(discussions []fetcher.Discussion) error {
	// Create output directory
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate Chroma CSS
	if err := g.generateChromaCSS(); err != nil {
		return fmt.Errorf("failed to generate chroma css: %w", err)
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
			Site        Config
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
			Site:        g.config,
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
			Site       Config
			Discussion fetcher.Discussion
		}{
			Site:       g.config,
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
		Site Config
		Tags map[string]int
	}{
		Site: g.config,
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
		Site        Config
		Tag         string
		Discussions []fetcher.Discussion
	}{
		Site:        g.config,
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
	rssPath := filepath.Join(g.outputDir, "rss.xml")
	file, err := os.Create(rssPath)
	if err != nil {
		return fmt.Errorf("failed to create rss.xml: %w", err)
	}
	defer file.Close()

	// Read the RSS template from file or use default
	rssTemplateContent, err := os.ReadFile(filepath.Join(g.templateDir, "../templates/rss.xml"))
	if err != nil {
		// Fallback to embedded RSS 2.0 template
		rssTemplateContent = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
    <channel>
        <title>{{.Site.Title}}</title>
        <description>{{.Site.Description}}</description>
        <link>{{.Site.URL}}</link>
        <lastBuildDate>{{.Updated}}</lastBuildDate>
        <language>{{.Site.Language | default "en"}}</language>
        <generator>Discussion Blog Generator</generator>
        {{range .Discussions}}
        <item>
            <title>{{.Title}}</title>
            <description>{{truncateHTML .Body 200}}</description>
            <link>{{.URL}}</link>
            <guid isPermaLink="true">{{.URL}}</guid>
            <pubDate>{{.CreatedAt.Format "Mon, 02 Jan 2006 15:04:05 MST"}}</pubDate>
        </item>
        {{end}}
    </channel>
</rss>`)
	}

	// For RSS XML, we need to use text/template to avoid HTML escaping of XML structure
	tmpl, err := texttemplate.New("rss").Funcs(texttemplate.FuncMap{
		"default": func(defaultValue, value interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"truncateHTML": func(s string, length int) string {
			// Convert markdown to HTML first
			htmlContent := blackfriday.Run([]byte(s))
			// Remove HTML tags
			re := regexp.MustCompile("<[^>]*>")
			plainText := re.ReplaceAllString(string(htmlContent), "")
			// Truncate to specified length
			if len(plainText) > length {
				plainText = plainText[:length] + "..."
			}
			
			// Handle line endings properly for XML
			plainText = strings.ReplaceAll(plainText, "\r\n", "\n")
			plainText = strings.ReplaceAll(plainText, "\r", "\n")
			
			// Clean the string to only include valid XML 1.0 characters
			// Valid XML 1.0 characters are: 
			// #x9 | #xA | #xD | [#x20-#xD7FF] | [#xE000-#xFFFD] | [#x10000-#x10FFFF]
			var cleaned strings.Builder
			for _, r := range plainText {
				if r == 0x09 || r == 0x0A || r == 0x0D || 
				   (r >= 0x20 && r <= 0xD7FF) || 
				   (r >= 0xE000 && r <= 0xFFFD) || 
				   (r >= 0x10000 && r <= 0x10FFFF) {
					cleaned.WriteRune(r)
				} else {
					// Replace invalid characters with a replacement character or skip them
					// For XML, we'll replace with a space or a question mark
					cleaned.WriteRune(' ')
				}
			}
			plainText = cleaned.String()
			
			// Escape XML special characters
			plainText = strings.ReplaceAll(plainText, "&", "&amp;")
			plainText = strings.ReplaceAll(plainText, "<", "&lt;")
			plainText = strings.ReplaceAll(plainText, ">", "&gt;")
			plainText = strings.ReplaceAll(plainText, "\"", "&quot;")
			plainText = strings.ReplaceAll(plainText, "'", "&apos;")
			
			// Replace newlines with XML character entities
			plainText = strings.ReplaceAll(plainText, "\n", "&#10;")
			
			return plainText
		},
		"markdown": func(s string) string {
			// Clean up line endings
			s = strings.ReplaceAll(s, "\r\n", "\n")
			s = strings.ReplaceAll(s, "\r", "\n")

			// Convert markdown to HTML with Chroma
			renderer := &ChromaRenderer{HTML: blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
				Flags: blackfriday.UseXHTML,
			})}
			extensions := blackfriday.CommonExtensions | blackfriday.AutoHeadingIDs | blackfriday.NoEmptyLineBeforeBlock
			html := blackfriday.Run([]byte(s), blackfriday.WithRenderer(renderer), blackfriday.WithExtensions(extensions))
			return string(html)
		},
		"trimBraces": func(s string) string {
			s = strings.TrimPrefix(s, "{")
			s = strings.TrimSuffix(s, "}")
			return s
		},
	}).Parse(string(rssTemplateContent))
	if err != nil {
		return fmt.Errorf("failed to parse RSS template: %w", err)
	}

	// Sort discussions by creation date (newest first) and take only the latest 10
	sortedDiscussions := make([]fetcher.Discussion, len(discussions))
	copy(sortedDiscussions, discussions)
	
	// Sort by CreatedAt in descending order (newest first)
	sort.Slice(sortedDiscussions, func(i, j int) bool {
		return sortedDiscussions[i].CreatedAt.After(sortedDiscussions[j].CreatedAt)
	})
	
	// Take only the first 10 discussions (newest ones)
	if len(sortedDiscussions) > 10 {
		sortedDiscussions = sortedDiscussions[:10]
	}

	// Get the most recent created time from the discussions we're including
	var updated time.Time
	for _, discussion := range sortedDiscussions {
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
			Language    string
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
			Language    string
		}{
			Title:       g.config.Site.Title,
			Description: g.config.Site.Description,
			URL:         g.config.Site.URL,
			Author:      g.config.Site.Author,
			Email:       g.config.Site.Email,
			Language:    g.config.Site.Language,
		},
		Updated:     updated.Format("Mon, 02 Jan 2006 15:04:05 MST"),
		Discussions: sortedDiscussions,
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute RSS template: %w", err)
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
		srcData, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Write to destination
		if err := os.WriteFile(destPath, srcData, 0644); err != nil {
			return err
		}

		return nil
	})
}

func (g *SiteGenerator) generateChromaCSS() error {
	// Write Chroma CSS to the source assets (public/styles),
	// so copyStaticAssets() will carry it into the output directory.
	publicStylesDir := filepath.Join("public", "styles")
	if err := os.MkdirAll(publicStylesDir, 0755); err != nil {
		return fmt.Errorf("failed to create public/styles directory: %w", err)
	}

	// Generate CSS for light theme (github)
	light := styles.Get("github")
	if light == nil {
		light = styles.Fallback
	}
	// Generate CSS for dark theme (github-dark for consistency)
	dark := styles.Get("github-dark")
	if dark == nil {
		dark = styles.Get("dracula")
		if dark == nil {
			dark = styles.Fallback
		}
	}

	formatter := html.New(html.WithClasses(true))

	var lightBuf bytes.Buffer
	if err := formatter.WriteCSS(&lightBuf, light); err != nil {
		return fmt.Errorf("failed to write light CSS: %w", err)
	}

	// Modify light theme CSS to add slight background difference and fix error styling
	lightCSS := lightBuf.String()
	// Add a subtle background color to .chroma and .bg classes in light theme
	lightCSS = strings.ReplaceAll(lightCSS, "/* Background */ .bg { background-color: #ffffff; }", "/* Background */ .bg { background-color: #f6f8fa; }")
	lightCSS = strings.ReplaceAll(lightCSS, "/* PreWrapper */ .chroma { background-color: #ffffff; }", "/* PreWrapper */ .chroma { background-color: #f6f8fa; }")
	// Modify error styling to be less intrusive
	lightCSS = strings.ReplaceAll(lightCSS, "/* Error */ .chroma .err { color: #f6f8fa; background-color: #82071e }", "/* Error */ .chroma .err { color: inherit; background-color: transparent }")

	var darkBuf bytes.Buffer
	if err := formatter.WriteCSS(&darkBuf, dark); err != nil {
		return fmt.Errorf("failed to write dark CSS: %w", err)
	}

	// Prefix dark CSS so it only applies when data-theme="dark"
	darkCSS := darkBuf.String()
	// Replace all occurrences of .chroma with [data-theme="dark"] .chroma
	darkCSS = strings.ReplaceAll(darkCSS, ".chroma", "[data-theme=\"dark\"] .chroma")
	// Replace all occurrences of .bg with [data-theme="dark"] .bg
	darkCSS = strings.ReplaceAll(darkCSS, ".bg", "[data-theme=\"dark\"] .bg")
	// Modify error styling in dark theme as well
	darkCSS = strings.ReplaceAll(darkCSS, "/* Error */ [data-theme=\"dark\"] .chroma .err { color: #f85149 }", "/* Error */ [data-theme=\"dark\"] .chroma .err { color: inherit; background-color: transparent }")

	var merged bytes.Buffer
	merged.WriteString("/* Chroma CSS (light: github) */\n")
	merged.WriteString(lightCSS)
	merged.WriteString("\n\n/* Chroma CSS (dark: github-dark) scoped to [data-theme=\"dark\"] */\n")
	merged.WriteString(darkCSS)

	// Write merged CSS to public/styles/chroma.css (overwrite if exists)
	chromaCSSPath := filepath.Join(publicStylesDir, "chroma.css")
	if err := os.WriteFile(chromaCSSPath, merged.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write chroma.css file: %w", err)
	}
	return nil
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
		Site       Config
		Discussion fetcher.Discussion
	}{
		Site:       g.config,
		Discussion: *aboutDiscussion,
	}

	// Execute the post template for about page
	if err := g.templates.ExecuteTemplate(file, "post.html", data); err != nil {
		return fmt.Errorf("failed to execute post template for about page: %w", err)
	}

	return nil
}
