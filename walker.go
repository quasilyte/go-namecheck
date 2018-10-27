package main

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

type walker struct {
	ctxt *context
	pkg  *packages.Package

	visit func(*[]*nameChecker, *ast.Ident)
}

func (w *walker) walkFunc(typ *ast.FuncType, body *ast.BlockStmt) {
	w.walkFieldList(&w.ctxt.checkers.param, typ.Params.List)
	// TODO(Quasilyte): add results scope and walk them?
	if body != nil {
		w.walkLocalNames(body)
	}
}

func (w *walker) walkNames(f *ast.File) {
	// TODO(Quasilyte): walk function param names
	// both in anonymous functions and in interface decls.

	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Recv != nil {
				w.walkFieldList(&w.ctxt.checkers.receiver, decl.Recv.List)
			}
			w.walkFunc(decl.Type, decl.Body)

		case *ast.GenDecl:
			w.walkGenDecl(&w.ctxt.checkers.global, decl)
		}
	}
}

func (w *walker) walkFieldList(checkers *[]*nameChecker, fields []*ast.Field) {
	for _, field := range fields {
		for _, id := range field.Names {
			w.visit(checkers, id)
		}
	}
}

func (w *walker) walkLocalNames(b *ast.BlockStmt) {
	ast.Inspect(b, func(x ast.Node) bool {
		switch x := x.(type) {
		case *ast.FuncLit:
			w.walkFunc(x.Type, x.Body)
			return false

		case *ast.AssignStmt:
			if x.Tok != token.DEFINE {
				return false
			}
			for _, lhs := range x.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok || w.pkg.TypesInfo.Defs[id] == nil {
					continue
				}
				w.visit(&w.ctxt.checkers.local, id)
			}
			return false

		case *ast.GenDecl:
			w.walkGenDecl(&w.ctxt.checkers.local, x)
			return false
		}

		return true
	})
}

func (w *walker) walkGenDecl(checkers *[]*nameChecker, decl *ast.GenDecl) {
	switch decl.Tok {
	case token.VAR, token.CONST:
		for _, spec := range decl.Specs {
			spec := spec.(*ast.ValueSpec)
			w.walkIdentList(checkers, spec.Names)
		}
	case token.TYPE:
		for _, spec := range decl.Specs {
			spec := spec.(*ast.TypeSpec)
			w.walkTypeExprNames(spec.Type)
		}
	}
}

func (w *walker) walkIdentList(checkers *[]*nameChecker, idents []*ast.Ident) {
	for _, id := range idents {
		w.visit(checkers, id)
	}
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
			w.visit(&w.ctxt.checkers.field, id)
		}
	}
}
