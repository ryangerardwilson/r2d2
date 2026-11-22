package health

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// The rows are judged without systemd, nebula, or a process. Only live.go,
// the edge, may import nebula or start a command.
func TestOnlyTheEdgeImportsNebulaAndExec(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") || name == "live.go" {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if strings.HasPrefix(path, "github.com/ryangerardwilson/nebula") || path == "os/exec" {
				t.Errorf("%s imports %s", name, path)
			}
		}
	}
}
