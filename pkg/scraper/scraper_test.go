package scraper

import (
	"strings"
	"testing"
)

func TestExtractText_Simple(t *testing.T) {
	html := `<html><body><h1>Title</h1><p>Hello world</p><ul><li>Item 1</li><li>Item 2</li></ul></body></html>`
	text := ExtractText(html)
	if !strings.Contains(text, "# Title") {
		t.Errorf("expected '# Title' in output, got: %s", text)
	}
	if !strings.Contains(text, "Hello world") {
		t.Errorf("expected 'Hello world' in output, got: %s", text)
	}
	if !strings.Contains(text, "- Item 1") {
		t.Errorf("expected '- Item 1' in output, got: %s", text)
	}
}

func TestExtractText_RemovesScripts(t *testing.T) {
	html := `<html><body><script>alert('xss')</script><p>Content</p><style>.foo{}</style></body></html>`
	text := ExtractText(html)
	if strings.Contains(text, "alert") {
		t.Errorf("expected script content to be removed, got: %s", text)
	}
	if strings.Contains(text, ".foo") {
		t.Errorf("expected style content to be removed, got: %s", text)
	}
	if !strings.Contains(text, "Content") {
		t.Errorf("expected 'Content' in output, got: %s", text)
	}
}

func TestExtractText_RemovesNav(t *testing.T) {
	html := `<html><body><nav><a href="/">Home</a></nav><main><p>Main content</p></main><footer>Footer</footer></body></html>`
	text := ExtractText(html)
	if strings.Contains(text, "Home") {
		t.Errorf("expected nav content to be removed, got: %s", text)
	}
	if strings.Contains(text, "Footer") {
		t.Errorf("expected footer content to be removed, got: %s", text)
	}
	if !strings.Contains(text, "Main content") {
		t.Errorf("expected 'Main content' in output, got: %s", text)
	}
}

func TestExtractTitle(t *testing.T) {
	html := `<html><head><title>My Page Title</title></head><body></body></html>`
	title := extractTitle(html)
	if title != "My Page Title" {
		t.Errorf("expected 'My Page Title', got '%s'", title)
	}
}
