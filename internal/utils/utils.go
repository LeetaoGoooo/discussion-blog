package utils

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"
)

// PreviewContent extracts a preview of the content
func PreviewContent(content string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(content, "")

	// Unescape HTML entities
	text = html.UnescapeString(text)

	// Limit to 200 characters
	if len(text) > 200 {
		return text[:200] + "..."
	}
	return text
}

// ToJSON converts data to JSON
func ToJSON(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// FormatDate formats a date
func FormatDate(date string) string {
	// For now, just return the date as is
	// In a real implementation, you might want to parse and reformat the date
	return date
}

// Slugify converts a string to a URL-friendly slug
func Slugify(text string) string {
	// Convert to lowercase
	slug := strings.ToLower(text)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters
	re := regexp.MustCompile(`[^a-z0-9\-]`)
	slug = re.ReplaceAllString(slug, "")

	// Replace multiple hyphens with single hyphen
	re = regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}