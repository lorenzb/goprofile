package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strconv"
)

// {
// 	f, err := os.Create("goprofile.prof")
// 	if err != nil {
// 		fmt.Println("Couldn't open goprofile.prof:", err)
// 		return
// 	}
// 	pprof.StartCPUProfile(f)
// 	defer pprof.StopCPUProfile()
// }
func newProfileStmt(proffile string) ast.Stmt {
	return &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.Ident{Name: "f"},
					&ast.Ident{Name: "err"},
				},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "os"},
							Sel: &ast.Ident{Name: "Create"},
						},
						Args: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: strconv.Quote(proffile),
							},
						},
					},
				},
			},
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  &ast.Ident{Name: "err"},
					Op: token.NEQ,
					Y:  &ast.Ident{Name: "nil"},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   &ast.Ident{Name: "fmt"},
									Sel: &ast.Ident{Name: "Println"},
								},
								Args: []ast.Expr{
									&ast.BasicLit{
										Kind:  token.STRING,
										Value: strconv.Quote(fmt.Sprintf("Couldn't open %s:", proffile)),
									},
									&ast.Ident{Name: "err"},
								},
							},
						},
						&ast.ReturnStmt{},
					},
				},
			},
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "pprof"},
						Sel: &ast.Ident{Name: "StartCPUProfile"},
					},
					Args: []ast.Expr{
						&ast.Ident{Name: "f"},
					},
				},
			},
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "pprof"},
						Sel: &ast.Ident{Name: "StopCPUProfile"},
					},
				},
			},
		},
	}
}

func newImportDecl(path string) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.IMPORT,
		Specs: []ast.Spec{
			&ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: path,
				},
			},
		},
	}
}

func isMain(fun *ast.FuncDecl) bool {
	return fun.Name.Name == "main" &&
		fun.Recv == nil &&
		fun.Type.Params.NumFields() == 0 &&
		fun.Type.Results == nil
}

func hasMain(file *ast.File) bool {
	var foundMain bool
	inspector := func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.File:
			if node.Name.Name != "main" {
				return false
			}
		case *ast.FuncDecl:
			if isMain(node) {
				foundMain = true
				return false
			}
		}

		return true
	}
	ast.Inspect(file, inspector)
	return foundMain
}

func hasImport(file *ast.File, path string) bool {
	var foundImport bool
	inspector := func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.ImportSpec:
			if node.Path.Value == path {
				foundImport = true
				return false
			}
		}
		return true
	}
	ast.Inspect(file, inspector)
	return foundImport
}

func instrument(file *ast.File, proffile string) {
	inspector := func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.File:
			var newDecls []ast.Decl

			if !hasImport(node, `"os"`) {
				newDecls = append(newDecls, newImportDecl(`"os"`))
			}

			if !hasImport(node, `"runtime/pprof"`) {
				newDecls = append(newDecls, newImportDecl(`"runtime/pprof"`))
			} else {
				fmt.Fprintln(os.Stderr, "Warning: runtime/pprof already imported. Maybe this program already supports profiling?")
			}

			newDecls = append(newDecls, node.Decls...)
			node.Decls = newDecls
		case *ast.FuncDecl:
			if isMain(node) {
				newBodyList := []ast.Stmt{newProfileStmt(proffile)}
				newBodyList = append(newBodyList, node.Body.List...)
				node.Body.List = newBodyList
			}
		}

		return true
	}
	ast.Inspect(file, inspector)
}
