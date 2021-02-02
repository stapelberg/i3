package tl8

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

func NewGoldmark() goldmark.Markdown {
	return NewGoldmarkWithOptions(nil, nil)
}

func NewGoldmarkWithOptions(parserOptions []parser.Option, rendererOptions []renderer.Option) goldmark.Markdown {
	return goldmark.New(
		// GFM is GitHub Flavored Markdown, which we need for tables, for
		// example.
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(append([]parser.Option{
			parser.WithAutoHeadingID(),
			// The Attribute option allows us to id, classes, and arbitrary
			// options on headings (for translation status).
			parser.WithAttribute(),
		}, parserOptions...)...),
		goldmark.WithRendererOptions(append([]renderer.Option{
			html.WithHardWraps(),
			html.WithXHTML(),
		}, rendererOptions...)...),
	)
}

type Heading struct {
	Line       int
	ID         string
	Translated string
	Text       string
}

type Section struct {
	Heading Heading
	Lines   []string
}

type Document struct {
	Version      string
	Sections     []Section
	SectionsByID map[string]Section
	Headings     []Heading
	HeadingsByID map[string]Heading
}

func Segment(source []byte) (*Document, error) {
	md := NewGoldmark()
	parser := md.Parser()
	rd := text.NewReader(source)
	root := parser.Parse(rd)

	// modeled after (go/token).File:
	var lineoffsets []int // lines contains the offset of the first character for each line (the first entry is always 0)

	processed := 0
	for {
		lineoffsets = append(lineoffsets, processed)
		idx := bytes.IndexByte(source[processed:], '\n')
		if idx == -1 {
			break
		}
		processed += idx + 1
	}

	doc := &Document{}

	var headings []Heading
	headingsByID := make(map[string]Heading)
	err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() == ast.KindHeading {
			var h Heading
			for _, attr := range n.Attributes() {
				b, ok := attr.Value.([]byte)
				if !ok {
					continue
				}
				val := string(b)
				switch string(attr.Name) {
				case "id":
					h.ID = val
				case "translated":
					h.Translated = val
				case "version":
					doc.Version = val
				}
			}
			if h.ID == "" {
				//return ast.WalkStop, fmt.Errorf("heading does not have id")
			}
			segments := n.Lines()
			first := segments.At(0)
			line := sort.Search(len(lineoffsets), func(i int) bool {
				return lineoffsets[i] > first.Start
			}) - 1
			if line < 0 {
				return ast.WalkStop, fmt.Errorf("BUG: could not find line offset for position %d", first.Start)
			}
			h.Line = line + 1
			h.Text = string(n.Text(source))
			headings = append(headings, h)
			headingsByID[h.ID] = h
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}

	var sections []Section
	sectionsByID := make(map[string]Section)
	// Split the document into lines, then segment the lines into sections based
	// on the headers.
	lines := strings.Split(string(source), "\n")
	for idx, h := range headings {
		end := len(lines) - 1
		if idx < len(headings)-1 {
			end = headings[idx+1].Line - 1
		}
		s := Section{
			Heading: h,
			Lines:   lines[h.Line:end],
		}
		sectionsByID[h.ID] = s
		sections = append(sections, s)
	}

	doc.Sections = sections
	doc.Headings = headings
	doc.HeadingsByID = headingsByID
	doc.SectionsByID = sectionsByID
	return doc, nil
}
