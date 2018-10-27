package main

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

type walker struct {
	pkg   *packages.Package
	visit func(*ast.Ident)
}

func (w *walker) walkNames(f *ast.File) {
	// TODO(Quasilyte): walk function param names
	// both in anonymous functions and in interface decls.

	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Recv != nil {
				w.walkFieldList(decl.Recv.List)
			}
			w.walkFieldList(decl.Type.Params.List)
			if decl.Type.Results != nil {
				w.walkFieldList(decl.Type.Results.List)
			}
			if decl.Body != nil {
				w.walkLocalNames(decl.Body)
			}

		case *ast.GenDecl:
			switch decl.Tok {
			case token.TYPE:
				for _, spec := range decl.Specs {
					spec := spec.(*ast.TypeSpec)
					w.walkTypeExprNames(spec.Type)
				}
			}
		}
	}
}

func (w *walker) walkFieldList(fields []*ast.Field) {
	for _, field := range fields {
		for _, id := range field.Names {
			w.visit(id)
		}
	}
}

func (w *walker) walkLocalNames(b *ast.BlockStmt) {
	ast.Inspect(b, func(x ast.Node) bool {
		switch x := x.(type) {
		case *ast.AssignStmt:
			if x.Tok != token.DEFINE {
				return false
			}
			for _, lhs := range x.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok || w.pkg.TypesInfo.Defs[id] == nil {
					continue
				}
				w.visit(id)
			}
			return false

		case *ast.GenDecl:
			// Decls always introduce new names.
			for _, spec := range x.Specs {
				spec, ok := spec.(*ast.ValueSpec)
				if !ok { // Ignore type/import specs
					return false
				}
				for _, id := range spec.Names {
					w.visit(id)
				}
			}
			return false
		}

		return true
	})
}

func (w *walker) walkTypeExprNames(e ast.Expr) {
	n, ok := e.(*ast.StructType)
	if !ok {
		return
	}
	for _, field := range n.Fields.List {
		if n, ok := field.Type.(*ast.StructType); ok {
			// Anonymous struct type. Need to visit its fields.
			w.walkTypeExprNames(n)
			continue
		}
		for _, id := range field.Names {
			w.visit(id)
		}
	}
}
