# Pure Static Blog Generator

A static site generator that uses GitHub Discussions as a CMS, built with Go.

## Features

- Uses GitHub Discussions as content management system
- Generates fully static HTML files
- Responsive design with Tailwind CSS
- Service Worker for offline capabilities
- RSS feed generation
- Search functionality
- Giscus comments integration
- Dark/light mode support

## Project Structure

```
.
├── .github/workflows/     # GitHub Actions workflows
├── api/                   # API handlers (for server mode)
├── cmd/                   # CLI commands
├── constants/             # Configuration and constants
├── content/               # Generated static content
├── core/                  # Core business logic
├── entities/              # Data structures
├── handlers/              # HTTP request handlers
├── internal/
│   ├── fetcher/           # Data fetching from GitHub
│   ├── generator/         # Static site generation
│   └── utils/             # Utility functions
├── public/                # Static assets
├── templates/             # HTML templates
├── static/                # Additional static files
├── config.yaml            # Configuration file
├── go.mod                 # Go module file
├── go.sum                 # Go checksum file
├── main.go                # Main entry point
└── README.md              # This file
```

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Node.js and npm (for Tailwind CSS)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/pure-static-blog.git
   cd pure-static-blog
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   npm install
   ```

3. Configure the application by editing `config.yaml`:
   ```yaml
   github:
     username: "your-github-username"
     repository: "your-discussion-repo"
     token: "your-github-token"
   
   site:
     title: "Your Blog Title"
     description: "Your blog description"
     url: "https://yourdomain.com"
   ```

### Generate Static Site

```bash
go run main.go generate
```

This will generate the static site in the `content/` directory.

### Local Development

```bash
# Generate CSS
npm run build

# Serve the site locally
go run main.go serve
```

The site will be available at `http://localhost:3000`.

## Deployment

### GitHub Pages

The project includes a GitHub Actions workflow for deploying to GitHub Pages:

1. Update `.github/workflows/deploy.yml` with your repository details
2. Push to the `main` branch
3. The site will be automatically deployed to GitHub Pages

### Other Platforms

The generated static files in the `content/` directory can be deployed to any static hosting service:
- Vercel
- Netlify
- AWS S3
- Firebase Hosting
- etc.

## Configuration

The `config.yaml` file contains all the necessary configuration options:

```yaml
github:
  username: "leetaogoooo"           # GitHub username
  repository: "discussion-blog"     # Repository name
  token: "your-github-token"        # GitHub personal access token

site:
  title: "Leetao's Blog"            # Site title
  description: "What I think, not only tech"  # Site description
  url: "http://www.leetao94.cn"     # Site URL
  author: "Leetao"                  # Author name
  email: "leetao94cn@gmail.com"     # Author email

build:
  outputDir: "content"              # Output directory for generated files
  postsPerPage: 10                  # Number of posts per page
```

## Customization

### Templates

HTML templates are located in the `templates/` directory:
- `index.html`: Homepage template
- `post.html`: Individual post template
- `tag.html`: Tag cloud template
- `search.html`: Search results template
- `rss.xml`: RSS feed template

### Styles

CSS is generated using Tailwind CSS:
1. Edit `constants/templates/css/input.css` to add custom styles
2. Run `npm run build` to generate the final CSS

### JavaScript

Custom JavaScript can be added to the templates directly or included as separate files in the `public/` directory.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.