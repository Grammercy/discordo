package markdown

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestRenderImage(t *testing.T) {
	src := []byte("![test](http://example.com/image.png)")
	parser := goldmark.DefaultParser()
	node := parser.Parse(text.NewReader(src))

	t.Run("Non-Kitty", func(t *testing.T) {
		t.Setenv("TERM", "xterm")
		r := NewRenderer(config.MessagesListTheme{})
		var buf bytes.Buffer
		r.Render(&buf, src, node)

		expected := "[Image: test]"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Kitty Cached", func(t *testing.T) {
		t.Setenv("TERM", "xterm-kitty")

		// Pre-populate cache
		url := "http://example.com/image.png"
		hash := sha256.Sum256([]byte(url))
		filename := hex.EncodeToString(hash[:]) + ".png"

		imagesDir := filepath.Join(consts.CacheDir(), "images")
		if err := os.MkdirAll(imagesDir, 0755); err != nil {
			t.Fatal(err)
		}

		imgPath := filepath.Join(imagesDir, filename)
		if err := os.WriteFile(imgPath, []byte("dummy image content"), 0644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(imgPath)

		r := NewRenderer(config.MessagesListTheme{})
		var buf bytes.Buffer
		r.Render(&buf, src, node)

		absPath, _ := filepath.Abs(imgPath)
		encodedPath := base64.StdEncoding.EncodeToString([]byte(absPath))
		expected := "\x1b_Ga=T,t=f;" + encodedPath + "\x1b\\"

		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Kitty Download", func(t *testing.T) {
		t.Setenv("TERM", "xterm-kitty")
		// New URL to avoid cache
		src2 := []byte("![test2](http://example.com/image2.png)")
		node2 := parser.Parse(text.NewReader(src2))

		// Ensure file doesn't exist
		url := "http://example.com/image2.png"
		hash := sha256.Sum256([]byte(url))
		filename := hex.EncodeToString(hash[:]) + ".png"
		imgPath := filepath.Join(consts.CacheDir(), "images", filename)
		os.Remove(imgPath) // Ensure it's gone

		r := NewRenderer(config.MessagesListTheme{})
		var buf bytes.Buffer
		r.Render(&buf, src2, node2)

		expected := "[Image Downloading...]"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})
}
