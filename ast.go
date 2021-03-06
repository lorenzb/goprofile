package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strconv"
)

// newProfileStmt returns an ast node equivalent to the following code:
// {
// 	f, err := os.Create("<proffile>")
// 	if err != nil {
// 		os.Stderr.WriteString("Couldn't open <proffile>: " + err.Error() + "\n")
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
									X: &ast.SelectorExpr{
										X:   &ast.Ident{Name: "os"},
										Sel: &ast.Ident{Name: "Stderr"},
									},
									Sel: &ast.Ident{Name: "WriteString"},
								},
								Args: []ast.Expr{
									&ast.BinaryExpr{
										Op: token.ADD,
										X: &ast.BinaryExpr{
											Op: token.ADD,
											X: &ast.BasicLit{
												Kind:  token.STRING,
												Value: strconv.Quote(fmt.Sprintf("Couldn't open %s: ", proffile)),
											},
											Y: &ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X:   &ast.Ident{Name: "err"},
													Sel: &ast.Ident{Name: "Error"},
												},
												Args: nil,
											},
										},
										Y: &ast.BasicLit{
											Kind:  token.STRING,
											Value: strconv.Quote("\n"),
										},
									},
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

// newImportDecls returns an ast node corresponding to an import
// declaration of the provided path
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

// hasImport determines whether the given file imports the package
// at path.
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

// instrument adds profiling code to the given file ast.
// If any of the packages required by the profiling code aren't present,
// instrument adds import declarations for them.
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
