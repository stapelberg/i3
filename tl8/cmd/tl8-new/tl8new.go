package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"translation/internal/tl8"
)

func tl8new1(fn string) error {
	source, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}

	doc, err := tl8.Segment(source)
	if err != nil {
		return err
	}

	lines := strings.Split(string(source), "\n")
	for idx, heading := range doc.Headings {
		if idx == 0 {
			continue // skip document title heading
		}
		line := lines[heading.Line-1]
		if strings.Contains(line, "translated=") {
			return fmt.Errorf("document already contains translated= markers")
		}
		line = strings.Replace(line, "}", ` translated="TODO"}`, 1)
		lines[heading.Line-1] = line
	}

	return ioutil.WriteFile(fn, []byte(strings.Join(lines, "\n")), 0644)
}

func tl8new() error {
	flag.Parse()
	if flag.NArg() != 1 {
		return fmt.Errorf("syntax: %s <markdown-file>", filepath.Base(os.Args[0]))
	}

	return tl8new1(flag.Arg(0))
}

func main() {
	if err := tl8new(); err != nil {
		log.Fatal(err)
	}
}
