// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command toolsvendor reapplies Go+ AST compatibility changes to the x/tools
// packages in cmd/vendor after "go mod vendor" replaces them.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type edit struct {
	path string
	old  string
	new  string
}

var edits = []edit{
	{
		"golang.org/x/tools/go/analysis/passes/unreachable/unreachable.go",
		`\tcase *ast.AssignStmt,
\t\t*ast.BadStmt,
\t\t*ast.DeclStmt,
\t\t*ast.DeferStmt,
\t\t*ast.EmptyStmt,
\t\t*ast.ExprStmt,`,
		`\tcase *ast.AssignStmt,
\t\t*ast.TryStmt,
\t\t*ast.BadStmt,
\t\t*ast.DeclStmt,
\t\t*ast.DeferStmt,
\t\t*ast.EmptyStmt,
\t\t*ast.ExprStmt,`,
	},
	{
		"golang.org/x/tools/go/analysis/passes/unreachable/unreachable.go",
		`\tcase *ast.AssignStmt,
\t\t*ast.BadStmt,
\t\t*ast.DeclStmt,
\t\t*ast.DeferStmt,
\t\t*ast.EmptyStmt,
\t\t*ast.GoStmt,`,
		`\tcase *ast.AssignStmt,
\t\t*ast.TryStmt,
\t\t*ast.BadStmt,
\t\t*ast.DeclStmt,
\t\t*ast.DeferStmt,
\t\t*ast.EmptyStmt,
\t\t*ast.GoStmt,`,
	},
	{
		"golang.org/x/tools/go/ast/astutil/enclosing.go",
		`\tcase *ast.AssignStmt:
\t\tchildren = append(children,
\t\t\ttok(n.TokPos, len(n.Tok.String())))

\tcase *ast.BasicLit:`,
		`\tcase *ast.AssignStmt:
\t\tchildren = append(children,
\t\t\ttok(n.TokPos, len(n.Tok.String())))

\tcase *ast.TryStmt:
\t\tchildren = append(children,
\t\t\ttok(n.Try, len("try")),
\t\t\ttok(n.TokPos, len(":=")))

\tcase *ast.BasicLit:`,
	},
	{
		"golang.org/x/tools/go/ast/astutil/enclosing.go",
		`\tcase *ast.AssignStmt:
\t\treturn "assignment"
\tcase *ast.BadDecl:`,
		`\tcase *ast.AssignStmt:
\t\treturn "assignment"
\tcase *ast.TryStmt:
\t\treturn "try statement"
\tcase *ast.BadDecl:`,
	},
	{
		"golang.org/x/tools/go/ast/astutil/rewrite.go",
		`\tcase *ast.AssignStmt:
\t\ta.applyList(n, "Lhs")
\t\ta.applyList(n, "Rhs")

\tcase *ast.GoStmt:`,
		`\tcase *ast.AssignStmt:
\t\ta.applyList(n, "Lhs")
\t\ta.applyList(n, "Rhs")

\tcase *ast.TryStmt:
\t\ta.applyList(n, "Lhs")
\t\ta.applyList(n, "Rhs")

\tcase *ast.GoStmt:`,
	},
	{
		"golang.org/x/tools/go/ast/edge/edge.go",
		`\tValueSpec_Names
\tValueSpec_Type
\tValueSpec_Values

\tmaxKind`,
		`\tValueSpec_Names
\tValueSpec_Type
\tValueSpec_Values
\tEnumDecl_Doc
\tEnumDecl_Name
\tEnumDecl_TypeParams
\tEnumDecl_Variants
\tEnumVariant_Comment
\tEnumVariant_Doc
\tEnumVariant_Fields
\tEnumVariant_Name
\tTryStmt_Lhs
\tTryStmt_Rhs

\tmaxKind`,
	},
	{
		"golang.org/x/tools/go/ast/edge/edge.go",
		`\tValueSpec_Names:       info[*ast.ValueSpec]("Names"),
\tValueSpec_Type:        info[*ast.ValueSpec]("Type"),
\tValueSpec_Values:      info[*ast.ValueSpec]("Values"),
}`,
		`\tValueSpec_Names:       info[*ast.ValueSpec]("Names"),
\tValueSpec_Type:        info[*ast.ValueSpec]("Type"),
\tValueSpec_Values:      info[*ast.ValueSpec]("Values"),
\tEnumDecl_Doc:          info[*ast.EnumDecl]("Doc"),
\tEnumDecl_Name:         info[*ast.EnumDecl]("Name"),
\tEnumDecl_TypeParams:   info[*ast.EnumDecl]("TypeParams"),
\tEnumDecl_Variants:     info[*ast.EnumDecl]("Variants"),
\tEnumVariant_Comment:   info[*ast.EnumVariant]("Comment"),
\tEnumVariant_Doc:       info[*ast.EnumVariant]("Doc"),
\tEnumVariant_Fields:    info[*ast.EnumVariant]("Fields"),
\tEnumVariant_Name:      info[*ast.EnumVariant]("Name"),
\tTryStmt_Lhs:           info[*ast.TryStmt]("Lhs"),
\tTryStmt_Rhs:           info[*ast.TryStmt]("Rhs"),
}`,
	},
	{
		"golang.org/x/tools/go/ast/inspector/typeof.go",
		`\tnTypeSwitchStmt
\tnUnaryExpr
\tnValueSpec
)`,
		`\tnTypeSwitchStmt
\tnUnaryExpr
\tnValueSpec
\tnEnumDecl
\tnEnumVariant
\tnTryStmt
)`,
	},
	{
		"golang.org/x/tools/go/ast/inspector/typeof.go",
		`\tcase *ast.ValueSpec:
\t\treturn 1 << nValueSpec
\t}
\treturn 0`,
		`\tcase *ast.ValueSpec:
\t\treturn 1 << nValueSpec
\tcase *ast.EnumDecl:
\t\treturn 1 << nEnumDecl
\tcase *ast.EnumVariant:
\t\treturn 1 << nEnumVariant
\tcase *ast.TryStmt:
\t\treturn 1 << nTryStmt
\t}
\treturn 0`,
	},
	{
		"golang.org/x/tools/go/ast/inspector/walk.go",
		`\t\twalkList(v, edge.GenDecl_Specs, n.Specs)

\tcase *ast.FuncDecl:`,
		`\t\twalkList(v, edge.GenDecl_Specs, n.Specs)

\tcase *ast.EnumDecl:
\t\tif n.Doc != nil {
\t\t\twalk(v, edge.EnumDecl_Doc, -1, n.Doc)
\t\t}
\t\twalk(v, edge.EnumDecl_Name, -1, n.Name)
\t\tif n.TypeParams != nil {
\t\t\twalk(v, edge.EnumDecl_TypeParams, -1, n.TypeParams)
\t\t}
\t\twalkList(v, edge.EnumDecl_Variants, n.Variants)

\tcase *ast.EnumVariant:
\t\tif n.Doc != nil {
\t\t\twalk(v, edge.EnumVariant_Doc, -1, n.Doc)
\t\t}
\t\twalk(v, edge.EnumVariant_Name, -1, n.Name)
\t\tif n.Fields != nil {
\t\t\twalk(v, edge.EnumVariant_Fields, -1, n.Fields)
\t\t}
\t\tif n.Comment != nil {
\t\t\twalk(v, edge.EnumVariant_Comment, -1, n.Comment)
\t\t}

\tcase *ast.FuncDecl:`,
	},
	{
		"golang.org/x/tools/go/cfg/builder.go",
		`\tcase *ast.DeclStmt:
\t\t// Treat each var ValueSpec as a separate statement.
\t\td := s.Decl.(*ast.GenDecl)
\t\tif d.Tok == token.VAR {`,
		`\tcase *ast.DeclStmt:
\t\t// Treat each var ValueSpec as a separate statement.
\t\td, ok := s.Decl.(*ast.GenDecl)
\t\tif !ok {
\t\t\tbreak // local enum or another declaration with no control-flow effect
\t\t}
\t\tif d.Tok == token.VAR {`,
	},
	{
		"golang.org/x/tools/internal/refactor/inline/calleefx.go",
		`\t\tcase *ast.DeclStmt:
\t\t\tdecl := n.Decl.(*ast.GenDecl)
\t\t\tfor _, spec := range decl.Specs {`,
		`\t\tcase *ast.DeclStmt:
\t\t\tdecl, ok := n.Decl.(*ast.GenDecl)
\t\t\tif !ok {
\t\t\t\treturn true // local enum declaration has no runtime effect
\t\t\t}
\t\t\tfor _, spec := range decl.Specs {`,
	},
	{
		"golang.org/x/tools/internal/refactor/inline/inline.go",
		`\t\tcase *ast.DeclStmt:
\t\t\tfor _, spec := range stmt.Decl.(*ast.GenDecl).Specs {
\t\t\t\tswitch spec := spec.(type) {
\t\t\t\tcase *ast.ValueSpec:
\t\t\t\t\tfor _, id := range spec.Names {
\t\t\t\t\t\tnames[id.Name] = true
\t\t\t\t\t}
\t\t\t\tcase *ast.TypeSpec:
\t\t\t\t\tnames[spec.Name.Name] = true
\t\t\t\t}
\t\t\t}`,
		`\t\tcase *ast.DeclStmt:
\t\t\tswitch decl := stmt.Decl.(type) {
\t\t\tcase *ast.GenDecl:
\t\t\t\tfor _, spec := range decl.Specs {
\t\t\t\t\tswitch spec := spec.(type) {
\t\t\t\t\tcase *ast.ValueSpec:
\t\t\t\t\t\tfor _, id := range spec.Names {
\t\t\t\t\t\t\tnames[id.Name] = true
\t\t\t\t\t\t}
\t\t\t\t\tcase *ast.TypeSpec:
\t\t\t\t\t\tnames[spec.Name.Name] = true
\t\t\t\t\t}
\t\t\t\t}
\t\t\tcase *ast.EnumDecl:
\t\t\t\tnames[decl.Name.Name] = true
\t\t\t\tfor _, variant := range decl.Variants {
\t\t\t\t\tnames[variant.Name.Name] = true
\t\t\t\t}
\t\t\t}`,
	},
	{
		"golang.org/x/tools/refactor/satisfy/find.go",
		`\tcase *ast.DeclStmt:
\t\td := s.Decl.(*ast.GenDecl)
\t\tif d.Tok == token.VAR { // ignore consts`,
		`\tcase *ast.DeclStmt:
\t\td, ok := s.Decl.(*ast.GenDecl)
\t\tif !ok {
\t\t\tbreak // local enum declaration has no assignment constraints
\t\t}
\t\tif d.Tok == token.VAR { // ignore consts`,
	},
	{
		"golang.org/x/tools/go/ast/inspector/walk.go",
		`\tcase *ast.AssignStmt:
\t\twalkList(v, edge.AssignStmt_Lhs, n.Lhs)
\t\twalkList(v, edge.AssignStmt_Rhs, n.Rhs)

\tcase *ast.GoStmt:`,
		`\tcase *ast.AssignStmt:
\t\twalkList(v, edge.AssignStmt_Lhs, n.Lhs)
\t\twalkList(v, edge.AssignStmt_Rhs, n.Rhs)

\tcase *ast.TryStmt:
\t\twalkList(v, edge.TryStmt_Lhs, n.Lhs)
\t\twalkList(v, edge.TryStmt_Rhs, n.Rhs)

\tcase *ast.GoStmt:`,
	},
	{
		"golang.org/x/tools/go/cfg/builder.go",
		`\t\t*ast.GoStmt,
\t\t*ast.EmptyStmt,
\t\t*ast.AssignStmt:`,
		`\t\t*ast.GoStmt,
\t\t*ast.EmptyStmt,
\t\t*ast.AssignStmt,
\t\t*ast.TryStmt:`,
	},
	{
		"golang.org/x/tools/internal/astutil/free/free.go",
		`\tcase *ast.AssignStmt:
\t\twalkSlice(v, n.Rhs)
\t\tif n.Tok == token.DEFINE {
\t\t\tv.shortVarDecl(n.Lhs)
\t\t} else {
\t\t\twalkSlice(v, n.Lhs)
\t\t}

\tcase *ast.LabeledStmt:`,
		`\tcase *ast.AssignStmt:
\t\twalkSlice(v, n.Rhs)
\t\tif n.Tok == token.DEFINE {
\t\t\tv.shortVarDecl(n.Lhs)
\t\t} else {
\t\t\twalkSlice(v, n.Lhs)
\t\t}

\tcase *ast.TryStmt:
\t\twalkSlice(v, n.Rhs)
\t\tv.shortVarDecl(n.Lhs)

\tcase *ast.LabeledStmt:`,
	},
	{
		"golang.org/x/tools/refactor/satisfy/find.go",
		`func (f *Finder) exprN(e ast.Expr) types.Type {
\ttyp := f.info.Types[e].Type.(*types.Tuple)
\tswitch e := e.(type) {`,
		`func (f *Finder) exprN(e ast.Expr) types.Type {
\ttyp := f.info.Types[e].Type
\ttuple, _ := typ.(*types.Tuple)
\tswitch e := e.(type) {`,
	},
	{
		"golang.org/x/tools/refactor/satisfy/find.go",
		`\tcase *ast.TypeAssertExpr:
\t\t// y, ok := x.(T)
\t\tf.typeAssert(f.expr(e.X), typ.At(0).Type())`,
		`\tcase *ast.TypeAssertExpr:
\t\t// y, ok := x.(T)
\t\tif tuple != nil {
\t\t\tf.typeAssert(f.expr(e.X), tuple.At(0).Type())
\t\t} else {
\t\t\tf.typeAssert(f.expr(e.X), typ)
\t\t}`,
	},
	{
		"golang.org/x/tools/refactor/satisfy/find.go",
		`\t\tdefault:
\t\t\t// y op= x
\t\t\tf.expr(s.Lhs[0])
\t\t\tf.expr(s.Rhs[0])
\t\t}

\tcase *ast.GoStmt:`,
		`\t\tdefault:
\t\t\t// y op= x
\t\t\tf.expr(s.Lhs[0])
\t\t\tf.expr(s.Rhs[0])
\t\t}

\tcase *ast.TryStmt:
\t\tvar rhsTuple types.Type
\t\tif len(s.Rhs) == 1 {
\t\t\trhsTuple = f.exprN(s.Rhs[0])
\t\t}
\t\tfor i, expr := range s.Lhs {
\t\t\tvar lhs, rhs types.Type
\t\t\tif rhsTuple == nil {
\t\t\t\trhs = f.expr(s.Rhs[i])
\t\t\t} else if _, ok := rhsTuple.(*types.Tuple); ok {
\t\t\t\trhs = f.extract(rhsTuple, i)
\t\t\t} else {
\t\t\t\trhs = rhsTuple
\t\t\t}
\t\t\tif id, ok := expr.(*ast.Ident); ok && id.Name != "_" {
\t\t\t\tif obj, ok := f.info.Defs[id]; ok {
\t\t\t\t\tlhs = obj.Type()
\t\t\t\t}
\t\t\t}
\t\t\tif lhs == nil {
\t\t\t\tlhs = f.expr(expr)
\t\t\t}
\t\t\tf.assign(lhs, rhs)
\t\t}
\t\tif rhsTuple == nil {
\t\t\tf.expr(s.Rhs[len(s.Rhs)-1])
\t\t}

\tcase *ast.GoStmt:`,
	},
}

func main() {
	vendor := flag.String("vendor", "vendor", "path to the vendor directory")
	flag.Parse()
	for _, edit := range edits {
		if err := apply(filepath.Join(*vendor, filepath.FromSlash(edit.path)), edit); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func apply(path string, edit edit) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(data)
	old := strings.ReplaceAll(edit.old, `\t`, "\t")
	new := strings.ReplaceAll(edit.new, `\t`, "\t")
	if strings.Contains(s, new) {
		return nil
	}
	if strings.Count(s, old) != 1 {
		return fmt.Errorf("%s: x/tools vendor edit no longer applies", path)
	}
	s = strings.Replace(s, old, new, 1)
	return os.WriteFile(path, []byte(s), 0o666)
}
