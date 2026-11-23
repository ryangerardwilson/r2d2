// Package archtest encodes r2d2's dependency rules so a violation fails go
// test: r2d2 requires the published c3po, nebula, and musica modules with no
// replace directive; a screen calls c3po in process and never runs the c3po
// command; and r2d2 only reads systemd units, it never changes one.
package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const root = "../.."

func TestGoModHasNoReplace(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(body), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == "replace" {
			t.Fatalf("go.mod:%d: %s", i+1, line)
		}
	}
}

// changeVerbs are the systemctl verbs that change a unit.
var changeVerbs = map[string]bool{
	"start": true, "stop": true, "restart": true, "reload": true,
	"enable": true, "disable": true, "kill": true, "reset-failed": true,
}

func TestCommandsStayInsideTheirLane(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "camera" || strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				inspectCommand(t, fset, call)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// reporter is the part of testing.T the rule needs.
type reporter interface {
	Errorf(format string, args ...any)
}

// inspectCommand reports exec.Command or exec.CommandContext that runs c3po,
// or that runs systemctl with a verb that changes a unit.
func inspectCommand(t reporter, fset *token.FileSet, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "exec" || (sel.Sel.Name != "Command" && sel.Sel.Name != "CommandContext") {
		return
	}
	args := call.Args
	if sel.Sel.Name == "CommandContext" && len(args) > 0 {
		args = args[1:]
	}
	words := literals(args)
	if len(words) == 0 {
		return
	}
	where := fset.Position(call.Pos())
	if filepath.Base(words[0]) == "c3po" {
		t.Errorf("%s: a screen runs c3po; call its exported functions in process", where)
	}
	if filepath.Base(words[0]) != "systemctl" {
		return
	}
	for _, word := range words[1:] {
		if changeVerbs[word] {
			t.Errorf("%s: systemctl %s changes a unit; r2d2 only reads units", where, word)
		}
	}
}

// literals returns the leading string literal arguments.
func literals(args []ast.Expr) []string {
	var out []string
	for _, arg := range args {
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			break
		}
		word, err := strconv.Unquote(lit.Value)
		if err != nil {
			break
		}
		out = append(out, word)
	}
	return out
}

func TestTheRuleCatchesAViolation(t *testing.T) {
	src := `package x
import ("context"; "os/exec")
func f(ctx context.Context) {
	exec.Command("c3po", "slack")
	exec.CommandContext(ctx, "systemctl", "--user", "restart", "tunnel.service")
	exec.Command("systemctl", "--user", "show", "tunnel.service")
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			inspectCommand(rec, fset, call)
		}
		return true
	})
	if rec.n != 2 {
		t.Fatalf("caught %d violations, want 2", rec.n)
	}
}

type recorder struct{ n int }

func (r *recorder) Errorf(string, ...any) { r.n++ }
