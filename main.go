package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"pure/internal/fetcher"
	"pure/internal/generator"
)

// Config represents the application configuration
type Config struct {
	Github struct {
		Owner string `mapstructure:"owner"`
		Repo  string `mapstructure:"repo"`
		Token string `mapstructure:"token"`
	} `mapstructure:"github"`
	Site struct {
		Title       string `mapstructure:"title"`
		URL         string `mapstructure:"url"`
		Description string `mapstructure:"description"`
		Author      string `mapstructure:"author"`
		Email       string `mapstructure:"email"`
		AboutID     int    `mapstructure:"about_id"`
		Giscus      struct {
			RepoID     string `mapstructure:"repo_id"`
			Category   string `mapstructure:"category"`
			CategoryID string `mapstructure:"category_id"`
		} `mapstructure:"giscus"`
		Favicon string `mapstructure:"favicon"`
	} `mapstructure:"site"`
}

var (
	cfgFile string
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "blog",
	Short: "A static site generator for blogs using GitHub Discussions as content source",
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate static site from GitHub Discussions",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Generating static site...")

		// 读取配置
		var config Config
		if err := viper.Unmarshal(&config); err != nil {
			log.Fatalf("Unable to decode into struct: %v", err)
		}

		// 获取 GitHub token，优先使用环境变量中的 GITHUB_TOKEN，如果不存在则使用配置文件中的 token
		githubToken := os.Getenv("GITHUB_TOKEN")
		if githubToken == "" {
			githubToken = config.Github.Token
		}

		// 获取数据
		fmt.Println("Fetching discussions from GitHub...")
		discussions, err := fetcher.FetchDiscussions(githubToken, config.Github.Owner, config.Github.Repo)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch discussions: %v\n", err)
			fmt.Println("Generating site with sample data...")
			// 使用示例数据
			discussions = getSampleDiscussions()
		}

		// 初始化生成器
		genConfig := generator.Config{
			Site: struct {
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
			}{
				Title:       config.Site.Title,
				URL:         config.Site.URL,
				Description: config.Site.Description,
				Author:      config.Site.Author,
				Email:       config.Site.Email,
				AboutID:     config.Site.AboutID,
				Giscus: struct {
					RepoID     string
					Category   string
					CategoryID string
				}{
					RepoID:     config.Site.Giscus.RepoID,
					Category:   config.Site.Giscus.Category,
					CategoryID: config.Site.Giscus.CategoryID,
				},
				Favicon: config.Site.Favicon,
			},
			Github: struct {
				Owner string
				Repo  string
			}{
				Owner: config.Github.Owner,
				Repo:  config.Github.Repo,
			},
		}

		// 创建输出目录
		outputPath := "./content"
		templatePath := "./templates/*.html"

		siteGen, err := generator.NewSiteGenerator(genConfig, templatePath, outputPath)
		if err != nil {
			log.Fatalf("Failed to create site generator: %v", err)
		}

		// 生成网站
		fmt.Println("Generating site files...")
		if err := siteGen.Generate(discussions); err != nil {
			log.Fatalf("Failed to generate site: %v", err)
		}

		fmt.Println("Site generated successfully in 'content' directory!")
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the generated site locally",
	Run: func(cmd *cobra.Command, args []string) {
		serveSite()
	},
}

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Generate and preview the site locally",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Generating static site...")

		// 读取配置
		var config Config
		if err := viper.Unmarshal(&config); err != nil {
			log.Fatalf("Unable to decode into struct: %v", err)
		}

		// 获取 GitHub token，优先使用环境变量中的 GITHUB_TOKEN，如果不存在则使用配置文件中的 token
		githubToken := os.Getenv("GITHUB_TOKEN")
		if githubToken == "" {
			githubToken = config.Github.Token
		}

		// 获取数据
		fmt.Println("Fetching discussions from GitHub...")
		discussions, err := fetcher.FetchDiscussions(githubToken, config.Github.Owner, config.Github.Repo)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch discussions: %v\n", err)
			fmt.Println("Generating site with sample data...")
			// 使用示例数据
			discussions = getSampleDiscussions()
		}

		// 初始化生成器
		genConfig := generator.Config{
			Site: struct {
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
			}{
				Title:       config.Site.Title,
				URL:         config.Site.URL,
				Description: config.Site.Description,
				Author:      config.Site.Author,
				Email:       config.Site.Email,
				AboutID:     config.Site.AboutID,
				Giscus: struct {
					RepoID     string
					Category   string
					CategoryID string
				}{
					RepoID:     config.Site.Giscus.RepoID,
					Category:   config.Site.Giscus.Category,
					CategoryID: config.Site.Giscus.CategoryID,
				},
				Favicon: config.Site.Favicon,
			},
			Github: struct {
				Owner string
				Repo  string
			}{
				Owner: config.Github.Owner,
				Repo:  config.Github.Repo,
			},
		}

		// 创建输出目录
		outputPath := "./content"
		templatePath := "./templates/*.html"

		siteGen, err := generator.NewSiteGenerator(genConfig, templatePath, outputPath)
		if err != nil {
			log.Fatalf("Failed to create site generator: %v", err)
		}

		// 生成网站
		fmt.Println("Generating site files...")
		if err := siteGen.Generate(discussions); err != nil {
			log.Fatalf("Failed to generate site: %v", err)
		}

		fmt.Println("Site generated successfully in 'content' directory!")

		// 启动本地服务器预览
		serveSite()
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(previewCmd)
}

func getSampleDiscussions() []fetcher.Discussion {
	// 创建一些示例讨论数据
	time1 := time.Date(2025, 9, 15, 10, 30, 0, 0, time.UTC)
	time2 := time.Date(2025, 9, 10, 14, 45, 0, 0, time.UTC)
	time3 := time.Date(2025, 9, 5, 9, 15, 0, 0, time.UTC)

	return []fetcher.Discussion{
		{
			ID:     "1",
			Number: 1,
			Title:  "Welcome to My Blog",
			Body:   "This is the first post on my new blog. I'm excited to share my thoughts and ideas with the world!",
			Author: "Leetao",
			Category: struct {
				ID   string
				Name string
			}{
				ID:   "1",
				Name: "General",
			},
			Labels: []struct {
				Name string
			}{
				{Name: "welcome"},
				{Name: "introduction"},
			},
			CreatedAt: time1,
			URL:       "http://www.leetao94.cn/posts/1",
		},
		{
			ID:     "2",
			Number: 2,
			Title:  "Understanding Go Concurrency",
			Body:   "Go's concurrency model is based on the idea of communicating sequential processes (CSP). In this post, we'll explore goroutines and channels...",
			Author: "Leetao",
			Category: struct {
				ID   string
				Name string
			}{
				ID:   "2",
				Name: "Technology",
			},
			Labels: []struct {
				Name string
			}{
				{Name: "go"},
				{Name: "concurrency"},
				{Name: "programming"},
			},
			CreatedAt: time2,
			URL:       "http://www.leetao94.cn/posts/2",
		},
		{
			ID:     "3",
			Number: 3,
			Title:  "Building a Static Site Generator",
			Body:   "In this tutorial, we'll build a static site generator using Go. We'll cover fetching content from GitHub Discussions and generating static HTML files...",
			Author: "Leetao",
			Category: struct {
				ID   string
				Name string
			}{
				ID:   "2",
				Name: "Technology",
			},
			Labels: []struct {
				Name string
			}{
				{Name: "go"},
				{Name: "web development"},
				{Name: "tutorial"},
			},
			CreatedAt: time3,
			URL:       "http://www.leetao94.cn/posts/3",
		},
	}
}

// serveSite 启动本地服务器提供content目录中的文件
func serveSite() {
	fmt.Println("Starting local server at http://localhost:8080")

	// 获取文件系统
	fs := http.FileServer(http.Dir("./content"))

	// 创建一个自定义处理器
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 如果请求的是根路径，则直接提供根目录的index.html
		if r.URL.Path == "/" {
			// 不再重写URL路径，直接使用根目录的index.html
		}
		// 否则使用文件服务器处理请求
		fs.ServeHTTP(w, r)
	})

	fmt.Println("Server is running at http://localhost:8080")
	fmt.Println("Press Ctrl+C to stop the server")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}