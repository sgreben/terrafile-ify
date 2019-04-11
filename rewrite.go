package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/hcl/hcl/token"
)

const (
	identModule  = "module"
	identSource  = "Source"
	identVersion = "Version"
)

type rewriter struct {
	replaceBy map[gitModuleReferenceKey]moduleReference
}

func (r *rewriter) RewriteFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%q: %v", path, err)
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read terraform Source %q: %v", path, err)
	}
	astRoot, err := hcl.Parse(string(b))
	if err != nil {
		return fmt.Errorf("parse terraform Source %q: %v", path, err)
	}
	objList, ok := astRoot.Node.(*ast.ObjectList)
	if !ok {
		return fmt.Errorf("unexpected top-level node type in %q: %T", path, astRoot.Node)
	}
	if err := r.RewriteAST(objList); err != nil {
		return fmt.Errorf("rewrite terraform AST %q: %v", path, err)
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, astRoot); err != nil {
		return fmt.Errorf("format terraform Source %q: v", err)
	}
	fmt.Fprintln(&buf)
	if err := ioutil.WriteFile(path, buf.Bytes(), info.Mode()); err != nil {
		return fmt.Errorf("write terraform Source %q: %v", path, err)
	}
	return nil
}

func (r *rewriter) RewriteAST(root *ast.ObjectList) error {
	for _, obj := range root.Items {
		switch {
		case firstObjectKeyTokenMatches(token.IDENT, identModule, obj): // ok
		default:
			continue
		}
		objVal, ok := obj.Val.(*ast.ObjectType)
		if !ok {
			continue
		}
		var sourceLiteral *ast.LiteralType
		var versionObjIndex *int
		var reference moduleReference
		for i, obj := range objVal.List.Items {
			literal, ok := obj.Val.(*ast.LiteralType)
			if !ok {
				continue
			}
			value, ok := literal.Token.Value().(string)
			if !ok {
				continue
			}
			switch {
			case firstObjectKeyTokenMatches(token.IDENT, identSource, obj):
				sourceLiteral = literal
				reference.Source = value
			case firstObjectKeyTokenMatches(token.IDENT, identVersion, obj): // ok
				i := i
				versionObjIndex = &i
				reference.Version = &value
			}
		}
		gitReference, ok := reference.Git()
		if !ok {
			continue
		}
		mappedReference, ok := r.replaceBy[gitReference.Key()]
		if !ok {
			continue
		}
		sourceLiteral.Token.Text = fmt.Sprintf("%q", mappedReference)
		if versionObjIndex != nil {
			i := *versionObjIndex
			objVal.List.Items = append(objVal.List.Items[:i], objVal.List.Items[i+1:]...)
		}
	}
	return nil
}

func firstObjectKeyTokenMatches(matchType token.Type, matchText string, o *ast.ObjectItem) bool {
	if len(o.Keys) == 0 {
		return false
	}
	token := o.Keys[0].Token
	return token.Type == matchType && token.Text == matchText
}
