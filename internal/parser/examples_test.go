package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExamplesParse(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.mar"))
	if err != nil {
		t.Fatalf("glob examples failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one example file")
	}

	for _, file := range files {
		file := file
		t.Run(filepath.Base(file), func(t *testing.T) {
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read example failed: %v", err)
			}
			if _, err := Parse(string(src)); err != nil {
				t.Fatalf("example should parse cleanly: %v", err)
			}
		})
	}
}
