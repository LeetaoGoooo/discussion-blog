package generator

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"pure/entities"
)

type NotesConfig struct {
	Title       string
	Description string
	URL         string
	Author      string
}

type NotesGenerator struct {
	config      NotesConfig
	templateDir string
	outputDir   string
	templates   *template.Template
}

func NewNotesGenerator(config NotesConfig, templateDir, outputDir string) (*NotesGenerator, error) {
	funcMap := template.FuncMap{
		"default": func(defaultValue, value interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
		// Pipe in Go templates passes value as LAST argument, so (length, s) not (s, length)
		"truncate": func(length int, s string) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"truncateHTML": notesTruncateHTML,
		"trimBraces": func(s string) string {
			s = strings.TrimPrefix(s, "{")
			s = strings.TrimSuffix(s, "}")
			return s
		},
		"markdown": func(s string) template.HTML {
			return template.HTML(s)
		},
		// Mark string as safe HTML to prevent auto-escaping
		"html": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	templates, err := template.New("").Funcs(funcMap).ParseGlob(templateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &NotesGenerator{
		config:      config,
		templateDir: templateDir,
		outputDir:   outputDir,
		templates:   templates,
	}, nil
}

func (g *NotesGenerator) Generate(notes []entities.Note) error {
	if len(notes) == 0 {
		return nil
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].CreatedAt.After(notes[j].CreatedAt)
	})

	if err := g.generateNotesPage(notes); err != nil {
		return fmt.Errorf("failed to generate notes page: %w", err)
	}

	if err := g.generateNotePages(notes); err != nil {
		return fmt.Errorf("failed to generate note pages: %w", err)
	}

	return nil
}

func (g *NotesGenerator) generateNotesPage(notes []entities.Note) error {
	notesDir := filepath.Join(g.outputDir, "memos")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return fmt.Errorf("failed to create memos directory: %w", err)
	}

	notesPath := filepath.Join(notesDir, "index.html")
	file, err := os.Create(notesPath)
	if err != nil {
		return fmt.Errorf("failed to create memos index.html: %w", err)
	}
	defer file.Close()

	data := struct {
		Site  NotesConfig
		Notes []entities.Note
	}{
		Site:  g.config,
		Notes: notes,
	}

	if err := g.templates.ExecuteTemplate(file, "notes.html", data); err != nil {
		return fmt.Errorf("failed to execute memos template: %w", err)
	}

	return nil
}

func (g *NotesGenerator) generateNotePages(notes []entities.Note) error {
	for i, note := range notes {
		noteDir := filepath.Join(g.outputDir, "memos", note.ID)
		if err := os.MkdirAll(noteDir, 0755); err != nil {
			return fmt.Errorf("failed to create note directory: %w", err)
		}

		notePath := filepath.Join(noteDir, "index.html")
		file, err := os.Create(notePath)
		if err != nil {
			return fmt.Errorf("failed to create note index.html: %w", err)
		}
		defer file.Close()

		var prevNote, nextNote *entities.Note
		if i > 0 {
			prevNote = &notes[i-1]
		}
		if i < len(notes)-1 {
			nextNote = &notes[i+1]
		}

		data := struct {
			Site      NotesConfig
			Note      entities.Note
			PrevNote  *entities.Note
			NextNote  *entities.Note
		}{
Site:     g.config,
		Note:     note,
		PrevNote: prevNote,
		NextNote: nextNote,
	}

		if err := g.templates.ExecuteTemplate(file, "note.html", data); err != nil {
			return fmt.Errorf("failed to execute note template: %w", err)
		}
	}

	return nil
}

func notesTruncateHTML(s string, length int) string {
	re := regexp.MustCompile("<[^>]*>")
	plain := re.ReplaceAllString(s, "")
	if len(plain) > length {
		plain = plain[:length] + "..."
	}
	return plain
}