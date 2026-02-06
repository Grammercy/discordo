// Package markdown defines a renderer for tview style tags.
package markdown

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/yuin/goldmark/ast"
	gmr "github.com/yuin/goldmark/renderer"
)

var downloading sync.Map

type Renderer struct {
	theme   config.MessagesListTheme
	isKitty bool

	listIx     *int
	listNested int
}

func NewRenderer(theme config.MessagesListTheme) *Renderer {
	isKitty := strings.Contains(os.Getenv("TERM"), "kitty")
	return &Renderer{theme: theme, isKitty: isKitty}
}

func (r *Renderer) AddOptions(opts ...gmr.Option) {}

func (r *Renderer) Render(w io.Writer, source []byte, node ast.Node) error {
	return ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := node.(type) {
		case *ast.Document:
		// noop
		case *ast.Heading:
			r.renderHeading(w, node, entering)
		case *ast.Text:
			r.renderText(w, node, entering, source)
		case *ast.FencedCodeBlock:
			r.renderFencedCodeBlock(w, node, entering, source)
		case *ast.AutoLink:
			r.renderAutoLink(w, node, entering, source)
		case *ast.Link:
			r.renderLink(w, node, entering)
		case *ast.Image:
			return r.renderImage(w, node, entering)
		case *ast.List:
			r.renderList(w, node, entering)
		case *ast.ListItem:
			r.renderListItem(w, entering)

		case *discordmd.Inline:
			r.renderInline(w, node, entering)
		case *discordmd.Mention:
			r.renderMention(w, node, entering)
		case *discordmd.Emoji:
			r.renderEmoji(w, node, entering)
		}

		return ast.WalkContinue, nil
	})
}

func (r *Renderer) renderHeading(w io.Writer, node *ast.Heading, entering bool) {
	if entering {
		io.WriteString(w, strings.Repeat("#", node.Level))
		io.WriteString(w, " ")
	} else {
		io.WriteString(w, "\n")
	}
}

func (r *Renderer) renderFencedCodeBlock(w io.Writer, node *ast.FencedCodeBlock, entering bool, source []byte) {
	io.WriteString(w, "\n")

	if entering {
		// language
		if l := node.Language(source); l != nil {
			io.WriteString(w, "|=> ")
			w.Write(l)
			io.WriteString(w, "\n")
		}

		// body
		lines := node.Lines()
		for i := range lines.Len() {
			line := lines.At(i)
			io.WriteString(w, "| ")
			w.Write(line.Value(source))
		}
	}
}

func (r *Renderer) renderAutoLink(w io.Writer, node *ast.AutoLink, entering bool, source []byte) {
	urlStyle := r.theme.URLStyle

	if entering {
		fg := urlStyle.GetForeground()
		bg := urlStyle.GetBackground()
		fmt.Fprintf(w, "[%s:%s]", fg, bg)
		w.Write(node.URL(source))
	} else {
		io.WriteString(w, "[-:-]")
	}
}

func (r *Renderer) renderLink(w io.Writer, node *ast.Link, entering bool) {
	urlStyle := r.theme.URLStyle
	if entering {
		fg := urlStyle.GetForeground()
		bg := urlStyle.GetBackground()
		fmt.Fprintf(w, "[%s:%s::%s]", fg, bg, node.Destination)
	} else {
		io.WriteString(w, "[-:-::-]")
	}
}

func (r *Renderer) renderImage(w io.Writer, node *ast.Image, entering bool) (ast.WalkStatus, error) {
	if !r.isKitty {
		if entering {
			io.WriteString(w, "[Image: ")
		} else {
			io.WriteString(w, "]")
		}
		return ast.WalkContinue, nil
	}

	if !entering {
		return ast.WalkContinue, nil
	}

	url := string(node.Destination)
	hash := sha256.Sum256(node.Destination)
	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".png"
	}
	filename := hex.EncodeToString(hash[:]) + ext
	path := filepath.Join(consts.CacheDir(), "images", filename)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		io.WriteString(w, "[Image Error]")
		return ast.WalkSkipChildren, nil
	}

	if _, err := os.Stat(path); err == nil {
		absPath, err := filepath.Abs(path)
		if err != nil {
			io.WriteString(w, "[Image Error]")
			return ast.WalkSkipChildren, nil
		}

		encodedPath := base64.StdEncoding.EncodeToString([]byte(absPath))
		io.WriteString(w, fmt.Sprintf("\x1b_Ga=T,t=f;%s\x1b\\", encodedPath))
		return ast.WalkSkipChildren, nil
	} else {
		if _, ok := downloading.LoadOrStore(url, true); !ok {
			go func() {
				defer downloading.Delete(url)

				resp, err := http.Get(url)
				if err != nil {
					return
				}
				defer resp.Body.Close()

				f, err := os.Create(path)
				if err != nil {
					return
				}
				defer f.Close()

				io.Copy(f, resp.Body)
			}()
		}
		io.WriteString(w, "[Image Downloading...]")
		return ast.WalkSkipChildren, nil
	}
}

func (r *Renderer) renderList(w io.Writer, node *ast.List, entering bool) {
	if node.IsOrdered() {
		r.listIx = &node.Start
	} else {
		r.listIx = nil
	}

	if entering {
		io.WriteString(w, "\n")
		r.listNested++
	} else {
		r.listNested--
	}
}

func (r *Renderer) renderListItem(w io.Writer, entering bool) {
	if entering {
		io.WriteString(w, strings.Repeat("  ", r.listNested-1))

		if r.listIx != nil {
			io.WriteString(w, strconv.Itoa(*r.listIx))
			io.WriteString(w, ". ")
			*r.listIx++
		} else {
			io.WriteString(w, "- ")
		}
	} else {
		io.WriteString(w, "\n")
	}
}

func (r *Renderer) renderText(w io.Writer, node *ast.Text, entering bool, source []byte) {
	if entering {
		w.Write(node.Segment.Value(source))
		switch {
		case node.HardLineBreak():
			io.WriteString(w, "\n\n")
		case node.SoftLineBreak():
			io.WriteString(w, "\n")
		}
	}
}
func (r *Renderer) renderInline(w io.Writer, node *discordmd.Inline, entering bool) {
	if start, end := attrToTag(node.Attr); entering && start != "" {
		io.WriteString(w, start)
	} else {
		io.WriteString(w, end)
	}
}

func (r *Renderer) renderMention(w io.Writer, node *discordmd.Mention, entering bool) {
	mentionStyle := r.theme.MentionStyle
	if entering {
		fg := mentionStyle.GetForeground()
		bg := mentionStyle.GetBackground()
		fmt.Fprintf(w, "[%s:%s:b]", fg, bg)

		switch {
		case node.Channel != nil:
			io.WriteString(w, "#"+node.Channel.Name)
		case node.GuildUser != nil:
			name := node.GuildUser.DisplayOrUsername()
			if member := node.GuildUser.Member; member != nil && member.Nick != "" {
				name = member.Nick
			}

			io.WriteString(w, "@"+name)
		case node.GuildRole != nil:
			io.WriteString(w, "@"+node.GuildRole.Name)
		}
	} else {
		io.WriteString(w, "[-:-:B]")
	}
}

func (r *Renderer) renderEmoji(w io.Writer, node *discordmd.Emoji, entering bool) {
	if entering {
		emojiStyle := r.theme.EmojiStyle
		fg := emojiStyle.GetForeground()
		bg := emojiStyle.GetBackground()
		fmt.Fprintf(w, "[%s:%s]", fg, bg)
		io.WriteString(w, ":"+node.Name+":")
	} else {
		io.WriteString(w, "[-:-]")
	}
}

func attrToTag(attr discordmd.Attribute) (string, string) {
	switch attr {
	case discordmd.AttrBold:
		return "[::b]", "[::B]"
	case discordmd.AttrItalics:
		return "[::i]", "[::I]"
	case discordmd.AttrUnderline:
		return "[::u]", "[::U]"
	case discordmd.AttrStrikethrough:
		return "[::s]", "[::S]"
	case discordmd.AttrMonospace:
		return "[::r]", "[::R]"
	default:
		return "", ""
	}
}
