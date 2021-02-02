package main

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {
	source := []byte(`# document {version="4_18"}

Introduction.

## first heading {#first}

Old explanation.

## second heading {#second}

Unchanged explanation.
`)
	tmp := t.TempDir()
	fn := filepath.Join(tmp, "userguide.markdown")
	if err := ioutil.WriteFile(fn, source, 0644); err != nil {
		t.Fatal(err)
	}

	if err := tl8new1(fn); err != nil {
		t.Fatal(err)
	}

	wantSource := []byte(`# document {version="4_18"}

Introduction.

## first heading {#first translated="TODO"}

Old explanation.

## second heading {#second translated="TODO"}

Unchanged explanation.
`)

	updatedSource, err := ioutil.ReadFile(fn)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantSource, updatedSource); diff != "" {
		t.Errorf("unexpected new translation update: diff (-want +got):\n%s", diff)
	}

}
