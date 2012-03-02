package main

import (
  "bytes"
  "flag"
  "fmt"
  "go/ast"
  "go/parser"
  "go/token"
  "log"
  "os"
  "strings"
)

var _ = log.Println
var exitCode int

func main() {
  flag.Parse()
  if flag.NArg() == 0 {
    doDir(".")
  } else {
    for _, name := range flag.Args() {
      // Is it a directory?
      if fi, err := os.Stat(name); err == nil && fi.IsDir() {
        doDir(name)
      } else {
        errorf("not a directory: %s", name)
      }
    }
  }
  os.Exit(exitCode)
}

// error formats the error to standard error, adding program
// identification and a newline
func errorf(format string, args ...interface{}) {
  fmt.Fprintf(os.Stderr, "deadcode: "+format+"\n", args...)
  exitCode = 2
}

func doDir(name string) {
  notests := func(info os.FileInfo) bool {
    if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") &&
      !strings.HasSuffix(info.Name(), "_test.go") {
      return true
    }
    return false
  }
  fs := token.NewFileSet()
  pkgs, err := parser.ParseDir(fs, name, notests, parser.Mode(0))
  if err != nil {
    errorf("%s", err)
    return
  }
  for _, pkg := range pkgs {
    doPackage(fs, pkg)
  }
}

var structs map[string]*ast.StructType = make(map[string]*ast.StructType)

func doPackage(fs *token.FileSet, pkg *ast.Package) {
  // First pass, find all the structs and save them
  for _, file := range pkg.Files {
    for _, decl := range file.Decls {
      switch n := decl.(type) {
      case *ast.GenDecl:
        // var, const, types
        for _, spec := range n.Specs {
          switch s := spec.(type) {
          case *ast.TypeSpec:
            switch st := s.Type.(type) {
            case *ast.StructType:
              fmt.Println("=== " + s.Name.Name + " ===")
              structs[s.Name.Name] = st
            }
          }
        }
      }
    }
  }

  out := bytes.NewBufferString("")
  // Second pass, generate the code.
  for _, file := range pkg.Files {
    for _, decl := range file.Decls {
      switch n := decl.(type) {
      case *ast.GenDecl:
        // var, const, types
        for _, spec := range n.Specs {
          switch s := spec.(type) {
          case *ast.TypeSpec:
            switch st := s.Type.(type) {
            case *ast.StructType:
              out.Reset()
              fmt.Println("=== " + s.Name.Name + " ===")
              genForType(st, out)
              fmt.Println(out.String())
            }
          }
        }
      }
    }
  }
}

func genForType(str *ast.StructType, out *bytes.Buffer) {
  out.WriteString("{")
  max := len(str.Fields.List)
  for n, field := range str.Fields.List {
    // Name
    if field.Tag != nil {
      for _, f := range strings.Split(field.Tag.Value, ",") {
      }
    } else {
      //TODO: do the following for all the field names.
      //      as in: struct { X, Y int32 }
      out.WriteString(`"` + field.Names[0].Name + `": {`)
    }
    // Type
    genType(field.Type, out)
    // Description
    out.WriteString(`, "description": "`)
    if field.Doc != nil {
      for _, c := range field.Doc.List {
        out.WriteString(c.Text)
      }
    } else if field.Comment != nil {
      for _, c := range field.Comment.List {
        out.WriteString(c.Text)
      }
    }
    // ---
    out.WriteString(`"`)
    out.WriteString(" }")
    if max > n+1 {
      out.WriteString(",")
    }
  }
  out.WriteString(" }")
}

func genType(expr ast.Expr, out *bytes.Buffer) {
  switch ft := expr.(type) {
  case *ast.Ident:
    if ft.Name == "int32" {
      out.WriteString(` "type": "integer" `)
    } else if ft.Name == "string" {
      out.WriteString(` "type": "string" `)
    } else if ft.Name == "bool" {
      out.WriteString(` "type": "boolean" `)
    } else if _, ok := structs[ft.Name]; ok {
      out.WriteString(` "type": "object", `)
      out.WriteString(` "properties":`)
      genForType(structs[ft.Name], out)
    }
  case *ast.ArrayType:
    out.WriteString(` "type": "array", `)
    out.WriteString(` "items": { `)
    genType(ft.Elt, out)
    out.WriteString(`}`)
  case *ast.StarExpr:
    genType(ft.X, out)
  default:
    out.WriteString(`"type": "??"`)
  }
}
