package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"translation/internal/tl8"

	"github.com/google/go-cmp/cmp"
)

// fn is e.g. userguide.markdown
func flag1(fn, oldPath string) error {
	path, err := filepath.Abs(fn)
	if err != nil {
		return err
	}
	currentSource, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	current, err := tl8.Segment(currentSource)
	if err != nil {
		return err
	}

	oldSource, err := ioutil.ReadFile(oldPath)
	if err != nil {
		return err
	}
	old, err := tl8.Segment(oldSource)
	if err != nil {
		return err
	}

	unchanged := make(map[string]bool)
	for _, current := range current.Sections {
		old, ok := old.SectionsByID[current.Heading.ID]
		if !ok {
			log.Printf("BUG: section %q not found in -old_path=%s", current.Heading.ID, oldPath)
			continue
		}
		diff := cmp.Diff(old.Lines, current.Lines)
		changed := diff != ""
		unchanged[current.Heading.ID] = !changed
		if changed {
			log.Printf("changed (-old +current):\n%s", diff)
		}
	}

	dir := filepath.Dir(path)
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if !fi.Mode().IsDir() || fi.Name() == "." || fi.Name() == ".." {
			continue
		}
		translationPath := filepath.Join(dir, fi.Name(), filepath.Base(fn))
		b, err := ioutil.ReadFile(translationPath)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Print(err)
			}
			continue
		}
		lines := strings.Split(string(b), "\n")
		log.Printf("processing translation %s", translationPath)
		translation, err := tl8.Segment(b)
		if err != nil {
			return err
		}
		for _, heading := range translation.Headings {
			if unchanged[heading.ID] && heading.Translated != "" {
				log.Printf("  updating heading %q (up-to-date)", heading.ID)
				lines[heading.Line-1] = translatedRe.ReplaceAllString(lines[heading.Line-1], `translated="`+current.Version+`"`)
			}
		}
		documentHeading := translation.Headings[0]
		lines[documentHeading.Line-1] = versionRe.ReplaceAllString(lines[documentHeading.Line-1], `version="`+current.Version+`"`)
		if err := ioutil.WriteFile(translationPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return err
		}
	}

	return nil
}

var (
	translatedRe = regexp.MustCompile(`translated="([^"]+)"`)
	versionRe    = regexp.MustCompile(`version="([^"]+)"`)
)

func tl8flag() error {
	var (
		oldPath = flag.String("old_path",
			"",
			"old version of the document")
	)
	flag.Parse()
	if flag.NArg() != 1 {
		return fmt.Errorf("syntax: %s <markdown-file>", filepath.Base(os.Args[0]))
	}
	if *oldPath == "" {
		return fmt.Errorf("-old_path is required")
	}
	fn := flag.Arg(0)
	if err := flag1(fn, *oldPath); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := tl8flag(); err != nil {
		log.Fatal(err)
	}
}
