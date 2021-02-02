package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"translation/internal/tl8"

	"github.com/google/renameio"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type translationStatusNode struct {
	ast.BaseBlock

	translatedVersion string
}

// Kind implements Node.Kind
func (n *translationStatusNode) Kind() ast.NodeKind {
	return kindTranslationStatus
}

// Dump implements Node.Dump
func (n *translationStatusNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

var kindTranslationStatus = ast.NewNodeKind("TranslationStatus")

type versionNode struct {
	ast.BaseBlock

	introduced string
}

// Kind implements Node.Kind
func (n *versionNode) Kind() ast.NodeKind {
	return kindVersionNode
}

// Dump implements Node.Dump
func (n *versionNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

var kindVersionNode = ast.NewNodeKind("VersionNode")

type tl8transformer struct{}

func modifyASTFromHeading(heading ast.Node) {
	tsn := &translationStatusNode{}
	vsn := &versionNode{}
	for _, attr := range heading.Attributes() {
		b, ok := attr.Value.([]byte)
		if !ok {
			continue
		}
		val := string(b)
		if string(attr.Name) == "introduced" {
			vsn.introduced = val
		}
		if string(attr.Name) == "translated" {
			tsn.translatedVersion = val
		}
		//log.Printf("heading attr, name=%s, value=%s", attr.Name, val)
	}

	if vsn.introduced != "" {
		// Adding a child will make it part of the heading HTML element
		// (e.g. <h1>)
		heading.AppendChild(heading, vsn)
	}

	if tsn.translatedVersion != "" {
		// Insert the TranslationStatus node after the heading:
		tsn.SetNextSibling(heading.NextSibling())
		heading.SetNextSibling(tsn)
	}
}

// Transform is called once per document.
func (t *tl8transformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		//log.Printf("ast.Walk(type=%v, kind=%v)", n.Type(), n.Kind())
		if n.Type() == ast.TypeDocument {
			return ast.WalkContinue, nil
		}
		if n.Kind() == ast.KindHeading && entering {
			modifyASTFromHeading(n)
		}
		return ast.WalkSkipChildren, nil
	})
}

func numericVersionToHuman(v string) string {
	return strings.ReplaceAll(v, "_", ".")
}

type tl8renderer struct {
	basename string
}

func (r *tl8renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindTranslationStatus, r.renderTranslationStatus)
	reg.Register(kindVersionNode, r.renderVersion)
}

func (r *tl8renderer) renderVersion(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	vsn := node.(*versionNode)
	fmt.Fprintf(w, `<span class="introduced">since i3 v%s</span>`, numericVersionToHuman(vsn.introduced))
	return ast.WalkContinue, nil
}

func (r *tl8renderer) renderTranslationStatus(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	tsn := node.(*translationStatusNode)

	// NOTE: Ideally we would link to a list of commits for the relevant file
	// between the last-translated version and the current version.
	// Unfortunately, GitHub does not seem to provide such a view.

	translatedVersion := numericVersionToHuman(tsn.translatedVersion)
	fmt.Fprintf(w, `<i>
Out-of-date! This section’s translation was last updated for i3 v%s
(<a href="https://github.com/i3/i3/commits/next/docs/%s">what changed?</a>) 
(<a href="https://github.com/i3/i3/edit/next/docs/%s">contribute</a>)
</i>
`,
		translatedVersion,
		r.basename,
		r.basename)
	return ast.WalkContinue, nil
}

func render1(fn, output string, headerTmpl, footerTmpl *template.Template) error {
	basename := strings.TrimSuffix(fn, filepath.Ext(fn))
	source, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}

	md := tl8.NewGoldmarkWithOptions(
		[]parser.Option{parser.WithASTTransformers(util.Prioritized(&tl8transformer{}, 1))},
		[]renderer.Option{renderer.WithNodeRenderers(util.Prioritized(&tl8renderer{
			basename: basename,
		}, 500))})

	out, err := renameio.TempFile("", output)
	if err != nil {
		return err
	}

	if headerTmpl != nil {
		doc, err := tl8.Segment(source)
		if err != nil {
			return err
		}

		documentHeading := doc.Headings[0]

		if err := headerTmpl.Execute(out, struct {
			Title string
		}{
			Title: documentHeading.Text,
		}); err != nil {
			return fmt.Errorf("rendering -header_template: %v", err)
		}
	}

	if err := md.Convert(source, out); err != nil {
		return err
	}

	if footerTmpl != nil {
		if err := footerTmpl.Execute(out, nil); err != nil {
			return fmt.Errorf("rendering -footer_template: %v", err)
		}
	}

	if err := out.CloseAtomicallyReplace(); err != nil {
		return err
	}
	return nil
}

func tl8render() error {
	var (
		header = flag.String("header_template",
			"",
			"path to a Go template file (https://golang.org/pkg/html/template/) containing the HTML that should be printed before converted markdown content")

		footer = flag.String("footer_template",
			"",
			"path to a Go template file (https://golang.org/pkg/html/template/) containing the HTML that should be printed after converted markdown content")

		outdir = flag.String("outdir",
			"",
			"output directory")
	)

	flag.Parse()
	if flag.NArg() < 1 {
		return fmt.Errorf("syntax: %s <markdown-file> [<markdown-file>...]", filepath.Base(os.Args[0]))
	}

	var headerTmpl, footerTmpl *template.Template
	if *header != "" {
		var err error
		headerTmpl, err = template.ParseFiles(*header)
		if err != nil {
			return err
		}
	}

	if *footer != "" {
		var err error
		footerTmpl, err = template.ParseFiles(*footer)
		if err != nil {
			return err
		}
	}

	for _, fn := range flag.Args() {
		basename := strings.TrimSuffix(filepath.Base(fn), ".markdown")
		output := filepath.Join(*outdir, basename+".html")
		if *outdir == "" {
			output = basename + ".html"
		}

		if err := render1(fn, output, headerTmpl, footerTmpl); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := tl8render(); err != nil {
		log.Fatal(err)
	}
}
