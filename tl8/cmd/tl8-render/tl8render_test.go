package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"testing"
	"translation/internal/tl8"

	"github.com/google/go-cmp/cmp"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

func TestRenderVersion(t *testing.T) {
	md := tl8.NewGoldmarkWithOptions(
		[]parser.Option{parser.WithASTTransformers(util.Prioritized(&tl8transformer{}, 1))},
		[]renderer.Option{renderer.WithNodeRenderers(util.Prioritized(&tl8renderer{}, 500))})
	source := []byte(`# heading {#heading_id introduced="4_16"}`)
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := `<h1 id="heading_id">heading<span class="introduced">since i3 v4.16</span></h1>
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected output: diff (-want +got):\n%s", diff)
	}
}

func TestRenderTranslation(t *testing.T) {
	md := tl8.NewGoldmarkWithOptions(
		[]parser.Option{parser.WithASTTransformers(util.Prioritized(&tl8transformer{}, 1))},
		[]renderer.Option{renderer.WithNodeRenderers(util.Prioritized(&tl8renderer{
			basename: "userguide",
		}, 500))})
	source := []byte(`# heading {#heading_id translated="4_17"}`)
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := `<h1 id="heading_id">heading</h1>
<i>
Out-of-date! This section’s translation was last updated for i3 v4.17
(<a href="https://github.com/i3/i3/commits/next/docs/userguide">what changed?</a>) 
(<a href="https://github.com/i3/i3/edit/next/docs/userguide">contribute</a>)
</i>
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected output: diff (-want +got):\n%s", diff)
	}
}

func TestRenderTemplate(t *testing.T) {
	headerTmpl, err := template.New("").Parse(`<html>
<head>
  <title>{{ .Title }}</title>
</head>
<body>
`)
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	fn := filepath.Join(tmpDir, "userguide.markdown")
	const userguideMarkdown = `# i3 User Guide
`
	if err := ioutil.WriteFile(fn, []byte(userguideMarkdown), 0644); err != nil {
		t.Fatal(err)
	}
	if err := render1(fn, headerTmpl, nil); err != nil {
		t.Fatal(err)
	}
	got, err := ioutil.ReadFile(filepath.Join(tmpDir, "userguide.html"))
	if err != nil {
		t.Fatal(err)
	}
	const want = `<html>
<head>
  <title>i3 User Guide</title>
</head>
<body>
<h1 id="i3-user-guide">i3 User Guide</h1>
`
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Fatalf("RenderTemplate: unexpected diff (-want +got):\n%s", diff)
	}
}
