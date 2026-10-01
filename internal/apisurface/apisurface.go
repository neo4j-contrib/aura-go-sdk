// Package apisurface renders a deterministic textual snapshot of a Go
// package's exported API: types, functions, methods, vars and consts. It is
// used in golden-file tests to detect accidental changes to a package's
// public contract between releases.
package apisurface

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Generate parses the Go package in dir (non-recursive) and returns a
// deterministic textual representation of its exported API surface.
func Generate(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	var files []*ast.File
	var pkgName string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return "", fmt.Errorf("parsing %s: %w", name, err)
		}
		files = append(files, f)
		pkgName = f.Name.Name
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no non-test .go files found in %s", dir)
	}

	docPkg, err := doc.NewFromFiles(fset, files, pkgName)
	if err != nil {
		return "", fmt.Errorf("building doc package for %s: %w", dir, err)
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package %s\n", pkgName)

	if len(docPkg.Consts) > 0 {
		buf.WriteString("\nCONST\n")
		for _, c := range docPkg.Consts {
			writeDecl(&buf, fset, c.Decl)
		}
	}

	if len(docPkg.Vars) > 0 {
		buf.WriteString("\nVAR\n")
		for _, v := range docPkg.Vars {
			writeDecl(&buf, fset, v.Decl)
		}
	}

	if len(docPkg.Funcs) > 0 {
		buf.WriteString("\nFUNC\n")
		for _, f := range sortedFuncs(docPkg.Funcs) {
			writeFunc(&buf, fset, f.Decl)
		}
	}

	types := make([]*doc.Type, len(docPkg.Types))
	copy(types, docPkg.Types)
	sort.Slice(types, func(i, j int) bool { return types[i].Name < types[j].Name })

	for _, t := range types {
		fmt.Fprintf(&buf, "\nTYPE %s\n", t.Name)
		stripUnexported(t.Decl)
		writeDecl(&buf, fset, t.Decl)

		for _, f := range sortedFuncs(t.Funcs) {
			writeFunc(&buf, fset, f.Decl)
		}
		for _, m := range sortedFuncs(t.Methods) {
			writeFunc(&buf, fset, m.Decl)
		}
	}

	return buf.String(), nil
}

func sortedFuncs(funcs []*doc.Func) []*doc.Func {
	out := make([]*doc.Func, len(funcs))
	copy(out, funcs)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// writeDecl prints a type/var/const GenDecl without its doc comments.
func writeDecl(buf *bytes.Buffer, fset *token.FileSet, decl *ast.GenDecl) {
	decl.Doc = nil
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			s.Doc, s.Comment = nil, nil
		case *ast.TypeSpec:
			s.Doc, s.Comment = nil, nil
		}
	}
	var out bytes.Buffer
	if err := format.Node(&out, fset, decl); err != nil {
		fmt.Fprintf(buf, "<error formatting decl: %v>\n", err)
		return
	}
	buf.Write(out.Bytes())
	buf.WriteString("\n\n")
}

// writeFunc prints a function/method signature, dropping its body, doc
// comment and receiver variable name so that implementation and cosmetic
// receiver renames don't show up as surface changes.
func writeFunc(buf *bytes.Buffer, fset *token.FileSet, decl *ast.FuncDecl) {
	decl.Doc = nil
	decl.Body = nil
	if decl.Recv != nil {
		for _, field := range decl.Recv.List {
			field.Names = nil
		}
	}
	var out bytes.Buffer
	if err := format.Node(&out, fset, decl); err != nil {
		fmt.Fprintf(buf, "<error formatting func: %v>\n", err)
		return
	}
	buf.Write(out.Bytes())
	buf.WriteString("\n\n")
}

// stripUnexported removes unexported struct fields and unexported interface
// methods from a type declaration, since those are not part of the public
// contract, and strips field-level doc/line comments so pure comment edits
// don't show up as surface changes.
func stripUnexported(decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		switch t := ts.Type.(type) {
		case *ast.StructType:
			t.Fields.List = filterFields(t.Fields.List)
		case *ast.InterfaceType:
			t.Methods.List = filterFields(t.Methods.List)
		}
	}
}

func filterFields(fields []*ast.Field) []*ast.Field {
	kept := fields[:0]
	for _, f := range fields {
		f.Doc, f.Comment = nil, nil
		if len(f.Names) == 0 {
			// Embedded field/interface: keyed by its type name.
			if ast.IsExported(embeddedName(f.Type)) {
				kept = append(kept, f)
			}
			continue
		}
		var names []*ast.Ident
		for _, n := range f.Names {
			if ast.IsExported(n.Name) {
				names = append(names, n)
			}
		}
		if len(names) > 0 {
			f.Names = names
			kept = append(kept, f)
		}
	}
	return kept
}

func embeddedName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return embeddedName(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name
	default:
		return ""
	}
}
