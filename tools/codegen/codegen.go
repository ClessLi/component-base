package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/format"
	"go/token"
	"go/types"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

var errCodeDocPrefix = `# 错误码
{{if eq .FullName .SimpleName}}{{.FullName}}{{else}}{{.FullName}} 系统(简称{{.SimpleName}}){{end}}错误码列表，由 {{.BackQuote}}{{.GenerateCommand}}{{.BackQuote}} 命令生成，不要对此文件做任何更改。
## 功能说明
如果返回结果中存在 {{.BackQuote}}code{{.BackQuote}} 字段，则表示调用 API 接口失败。例如：
{{.BackQuote}}{{.BackQuote}}{{.BackQuote}}json
{
  "code": 100101,
  "message": "Database error"
}
{{.BackQuote}}{{.BackQuote}}{{.BackQuote}}
上述返回中 {{.BackQuote}}code{{.BackQuote}} 表示错误码，{{.BackQuote}}message{{.BackQuote}} 表示该错误的具体信息。每个错误同时也对应一个 HTTP 状态码，比如上述错误码对应了 HTTP 状态码 500(Internal Server Error)。
## 错误码列表
{{.SimpleName}} 系统支持的错误码列表如下：

| Identifier | Code | HTTP Code | Description | Reference |
| ---------- | ---- | --------- | ----------- | --------- |
`

var (
	typeNames         = flag.String("type", "", "comma-separated list of type names; must be set")
	output            = flag.String("output", "", "output file name for error codes; default srcdir/<type>_generated.go")
	wrapperOutput     = flag.String("wrapper-output", "", "output file name for wrapper functions; default srcdir/errors_generated.go")
	trimprefix        = flag.String("trimprefix", "", "trim the `prefix` from the generated constant names")
	buildTags         = flag.String("tags", "", "comma-separated list of build tags to apply")
	docOutput         = flag.String("doc-output", "", "also generate error code documentation to this file")
	projectFullName   = flag.String("fullname", "", "the project full name; must be set")
	projectSimpleName = flag.String("simplename", "", "the project simple name; default same as `fullname`")
	namespace         = flag.String("namespace", "", "the error code namespace for registration; default same as `simplename` in lowercase")
	wrapper           = flag.Bool("wrapper", false, "if true generate wrapper functions for errors package")
)

// docPath stores the documentation file path for reference link generation.
// It's set when -doc-output is specified or when -doc is used with -output.
var docPath string

type TemplateInfo struct {
	BackQuote, FullName, SimpleName, Namespace string
	GenerateCommand                            string
}

// Usage is a replacement usage function for the flags package.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of codegen:\n")
	fmt.Fprintf(os.Stderr, "\tcodegen [flags] -type T -fullname projectFullName -simplename projectSimpleName [directory]\n")
	fmt.Fprintf(os.Stderr, "\tcodegen [flags] -type T -fullname projectFullName -simplename projectSimpleName files... # Must be a single package\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("codegen: ")
	flag.Usage = Usage
	flag.Parse()
	if len(*typeNames) == 0 || len(*projectFullName) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	// project simple name default same as full name
	if len(*projectSimpleName) == 0 {
		*projectSimpleName = *projectFullName
	}

	// namespace default same as simple name in lowercase
	if len(*namespace) == 0 {
		*namespace = strings.ToLower(*projectSimpleName)
	}

	pTypes := strings.Split(*typeNames, ",")
	var tags []string
	if len(*buildTags) > 0 {
		tags = strings.Split(*buildTags, ",")
	}

	// We accept either one directory or a list of files. Which do we have?
	args := flag.Args()
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}

	// Parse the package once.
	var dir string
	g := Generator{
		trimPrefix: *trimprefix,
		typeNames:  pTypes,
	}
	// TODO(suzmue): accept other patterns for packages (directories, list of files, import paths, etc).
	if len(args) == 1 && isDirectory(args[0]) {
		dir = args[0]
	} else {
		if len(tags) != 0 {
			log.Fatal("-tags option applies only to directories, not when files are specified")
		}
		dir = filepath.Dir(args[0])
	}

	g.parsePackage(args, tags)

	// Determine output file paths
	outputName := *output
	if outputName == "" {
		absDir, _ := filepath.Abs(dir)
		baseName := fmt.Sprintf("%s_generated.go", strings.ReplaceAll(filepath.Base(absDir), "-", "_"))
		if len(flag.Args()) == 1 {
			baseName = fmt.Sprintf(
				"%s_generated.go",
				strings.ReplaceAll(filepath.Base(strings.TrimSuffix(flag.Args()[0], ".go")), "-", "_"),
			)
		}
		outputName = filepath.Join(dir, strings.ToLower(baseName))
	}

	// Set docPath for reference link generation BEFORE generating code
	if *docOutput != "" {
		docPath = *docOutput
	}

	// Parse values FIRST (needed by both doc and code generation)
	g.parseValues(g.typeNames[0])

	// Generate doc file FIRST if -doc-output is specified (to calculate docLine for each error code)
	if *docOutput != "" {
		g.generateDocs(g.typeNames[0])
		docSrc := g.buf.Bytes()

		err := os.WriteFile(*docOutput, docSrc, 0o600)
		if err != nil {
			log.Fatalf("writing doc output: %s", err)
		}
		g.buf.Reset()
	}

	// Generate code file (uses docLine calculated during doc generation)
	g.generateCode()
	src := g.format()

	// Write code file
	err := os.WriteFile(outputName, src, 0o600)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}

	// Generate wrapper file if -wrapper is specified
	if *wrapper {
		// Reset buffer before generating wrapper
		g.buf.Reset()

		// Resolve wrapper output path relative to source directory if it's a relative path
		wrapperPath := *wrapperOutput
		if wrapperPath == "" {
			// Default: errors_generated.go in source directory
			wrapperPath = filepath.Join(dir, "errors_generated.go")
		} else if !filepath.IsAbs(wrapperPath) {
			wrapperPath = filepath.Join(dir, wrapperPath)
		}

		g.generateWrapper()
		wrapperSrc := g.format()

		err := os.WriteFile(wrapperPath, wrapperSrc, 0o600)
		if err != nil {
			log.Fatalf("writing wrapper output: %s", err)
		}
	}
}

// isDirectory reports whether the named file is a directory.
func isDirectory(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		log.Fatal(err)
	}

	return info.IsDir()
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	buf       bytes.Buffer // Accumulated output.
	pkg       *Package     // Package we are scanning.
	typeNames []string     // Type names to generate code for.
	values    []Value      // Parsed values from source code.

	trimPrefix string
}

// Printf like fmt.Printf, but add the string to g.buf.
func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

// File holds a single parsed file and associated data.
type File struct {
	pkg  *Package  // Package to which this file belongs.
	file *ast.File // Parsed AST.
	// These fields are reset for each type being generated.
	typeName string  // Name of the constant type.
	values   []Value // Accumulator for constant values of that type.

	trimPrefix string
}

// Package defines options for package.
type Package struct {
	name  string
	defs  map[*ast.Ident]types.Object
	files []*File
	fset  *token.FileSet
}

// parsePackage analyzes the single package constructed from the patterns and tags.
// parsePackage exits if there is an error.
func (g *Generator) parsePackage(patterns []string, tags []string) {
	cfg := &packages.Config{
		// nolint: staticcheck
		Mode: packages.LoadSyntax,
		// TODO: Need to think about constants in test files. Maybe write type_string_test.go
		// in a separate pass? For later.
		Tests:      false,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(tags, " "))},
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("error: %d packages found", len(pkgs))
	}
	g.addPackage(pkgs[0])
}

// addPackage adds a type checked Package and its syntax files to the generator.
func (g *Generator) addPackage(pkg *packages.Package) {
	g.pkg = &Package{
		name:  pkg.Name,
		defs:  pkg.TypesInfo.Defs,
		files: make([]*File, len(pkg.Syntax)),
		fset:  pkg.Fset,
	}

	for i, file := range pkg.Syntax {
		g.pkg.files[i] = &File{
			file:       file,
			pkg:        g.pkg,
			trimPrefix: g.trimPrefix,
		}
	}
}

// generateCode produces the register calls for all types.
func (g *Generator) generateCode() {
	codegencmd := "codegen"
	for _, arg := range os.Args[1:] {
		if strings.Contains(arg, " ") {
			arg = "'" + arg + "'"
		}
		codegencmd = strings.Join([]string{codegencmd, arg}, " ")
	}

	// Print the header and package clause.
	g.Printf("// Copyright 2021 Cless Li <ClessLee@hotmail.com>. All rights reserved.\n")
	g.Printf("// Use of this source code is governed by a MIT style\n")
	g.Printf("// license that can be found in the LICENSE file.\n")
	g.Printf("\n")
	g.Printf("// Code generated by `%s`; DO NOT EDIT.\n", codegencmd)
	g.Printf("\n")
	g.Printf("package %s", g.pkg.name)
	g.Printf("\n")
	g.Printf("\n")
	g.Printf("import (\n")
	g.Printf("\t\"github.com/ClessLi/component-base/pkg/errors\"\n")
	g.Printf(")\n")
	g.Printf("\n")
	// Generate namespace constant
	g.Printf("// Namespace is the error code namespace for this package.\n")
	g.Printf("const Namespace = %q\n", *namespace)
	g.Printf("\n")

	// Generate init function for each type (values already parsed)
	for _, typeName := range g.typeNames {
		g.generate(typeName)
	}
}

// parseValues extracts values from source code for the given type.
func (g *Generator) parseValues(typeName string) {
	values := make([]Value, 0, 100)
	for _, file := range g.pkg.files {
		// Set the state for this run of the walker.
		file.typeName = typeName
		file.values = nil
		if file.file != nil {
			ast.Inspect(file.file, file.genDecl)
			values = append(values, file.values...)
		}
	}

	if len(values) == 0 {
		log.Fatalf("no values defined for type %s", typeName)
	}

	g.values = values
}

// generate produces the register calls for the named type.
func (g *Generator) generate(typeName string) {
	if len(g.values) == 0 {
		log.Fatalf("no values defined for type %s", typeName)
	}
	// Generate code that will fail if the constants change value.
	g.Printf("// init registers error codes defined in this source code to `github.com/ClessLi/component-base/pkg/errors`\n")
	g.Printf("func init() {\n")
	for _, v := range g.values {
		code, description := v.ParseComment()
		ref := v.ParseReference()
		g.Printf("\terrors.Register(errors.DefaultCoder(%s, %s, %q, %q, Namespace))\n", v.originalName, code, description, ref)
	}
	g.Printf("}\n")
}

// generateWrapper produces wrapper functions for errors package.
func (g *Generator) generateWrapper() {
	if len(g.values) == 0 {
		log.Fatalf("no values defined for wrapper generation")
	}

	// Print header
	g.Printf("// Copyright 2021 Cless Li <ClessLee@hotmail.com>. All rights reserved.\n")
	g.Printf("// Use of this source code is governed by a MIT style\n")
	g.Printf("// license that can be found in the LICENSE file.\n")
	g.Printf("\n")
	g.Printf("// Code generated by `codegen -wrapper`; DO NOT EDIT.\n")
	g.Printf("\n")
	g.Printf("package %s", g.pkg.name)
	g.Printf("\n")
	g.Printf("\n")
	g.Printf("import (\n")
	g.Printf("\t\"github.com/ClessLi/component-base/pkg/errors\"\n")
	g.Printf(")\n")
	g.Printf("\n")

	// Generate wrapper functions that mirror errors package API
	// but auto-inject the namespace constant

	// WithCode wrapper
	g.Printf("// WithCode creates an error with code and auto-injected namespace.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\terr := WithCode(ErrPageNotFound, \"user %%s not found\", \"12345\")\n")
	g.Printf("//\tfmt.Println(err)          // Output: Check the URL path and try again\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", err) // Output: user 12345 not found - #0 (codeexample:100006) Check the URL path and try again\n")
	g.Printf("//\n")
	g.Printf("func WithCode(code int, format string, args ...interface{}) error {\n")
	g.Printf("\treturn errors.WithCode(Namespace, code, format, args...)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// WrapC wrapper
	g.Printf("// WrapC wraps an error with code and auto-injected namespace.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\terr := WrapC(originalErr, ErrDatabase, \"failed to connect to %%s\", \"db\")\n")
	g.Printf("//\tfmt.Println(err)          // Output: Contact system administrator to resolve database connectivity issue\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", err) // Output: failed to connect to db - #0 (codeexample:100007) Contact system administrator to resolve database connectivity issue\n")
	g.Printf("//\n")
	g.Printf("func WrapC(err error, code int, format string, args ...interface{}) error {\n")
	g.Printf("\treturn errors.WrapC(err, Namespace, code, format, args...)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// Wrap wrapper
	g.Printf("// Wrap returns an error annotating err with a stack trace\n")
	g.Printf("// at the point Wrap is called, and the supplied message.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\terr := Wrap(originalErr, \"failed to connect to database\")\n")
	g.Printf("//\tfmt.Println(err)          // Output: failed to connect to database\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", err) // Output: failed to connect to database - stack trace details\n")
	g.Printf("//\n")
	g.Printf("func Wrap(err error, message string) error {\n")
	g.Printf("\treturn errors.Wrap(err, message)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// Wrapf wrapper
	g.Printf("// Wrapf returns an error annotating err with a stack trace\n")
	g.Printf("// at the point Wrapf is called, and the format specifier.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\terr := Wrapf(originalErr, \"failed to write to %%s\", \"/var/log\")\n")
	g.Printf("//\tfmt.Println(err)          // Output: failed to write to /var/log\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", err) // Output: failed to write to /var/log - stack trace details\n")
	g.Printf("//\n")
	g.Printf("func Wrapf(err error, format string, args ...interface{}) error {\n")
	g.Printf("\treturn errors.Wrapf(err, format, args...)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// New wrapper
	g.Printf("// New returns an error with the supplied message.\n")
	g.Printf("// New also records the stack trace at the point it was called.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\terr := New(\"invalid email format\")\n")
	g.Printf("//\tfmt.Println(err)          // Output: invalid email format\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", err) // Output: invalid email format - stack trace details\n")
	g.Printf("//\n")
	g.Printf("func New(message string) error {\n")
	g.Printf("\treturn errors.New(message)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// Errorf wrapper
	g.Printf("// Errorf formats according to a format specifier and returns the string\n")
	g.Printf("// as a value that satisfies error.\n")
	g.Printf("// Errorf also records the stack trace at the point it was called.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\terr := Errorf(\"connection timeout after %%d seconds\", 30)\n")
	g.Printf("//\tfmt.Println(err)          // Output: connection timeout after 30 seconds\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", err) // Output: connection timeout after 30 seconds - stack trace details\n")
	g.Printf("//\n")
	g.Printf("func Errorf(format string, args ...interface{}) error {\n")
	g.Printf("\treturn errors.Errorf(format, args...)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// WithStack wrapper
	g.Printf("// WithStack annotates err with a stack trace.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\terr := WithStack(originalErr)\n")
	g.Printf("//\tfmt.Println(err)          // Output: original error message\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", err) // Output: original error message - stack trace details\n")
	g.Printf("//\n")
	g.Printf("func WithStack(err error) error {\n")
	g.Printf("\treturn errors.WithStack(err)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// WithMessage wrapper
	g.Printf("// WithMessage annotates err with a new message.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\terr := WithMessage(originalErr, \"failed to parse request body\")\n")
	g.Printf("//\tfmt.Println(err)          // Output: failed to parse request body\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", err) // Output: failed to parse request body - stack trace details\n")
	g.Printf("//\n")
	g.Printf("func WithMessage(err error, message string) error {\n")
	g.Printf("\treturn errors.WithMessage(err, message)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// WithMessagef wrapper
	g.Printf("// WithMessagef annotates err with a formatted message.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\terr := WithMessagef(originalErr, \"failed to encode %%s response\", \"JSON\")\n")
	g.Printf("//\tfmt.Println(err)          // Output: failed to encode JSON response\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", err) // Output: failed to encode JSON response - stack trace details\n")
	g.Printf("//\n")
	g.Printf("func WithMessagef(err error, format string, args ...interface{}) error {\n")
	g.Printf("\treturn errors.WithMessagef(err, format, args...)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// Cause wrapper
	g.Printf("// Cause returns the underlying cause of the error.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\tcause := Cause(wrappedErr)\n")
	g.Printf("//\tfmt.Println(cause)          // Output: root cause message\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", cause) // Output: root cause message - stack trace details\n")
	g.Printf("//\n")
	g.Printf("func Cause(err error) error {\n")
	g.Printf("\treturn errors.Cause(err)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// Aggregate error functions

	// NewAggregate wrapper
	g.Printf("// NewAggregate converts a slice of errors into an Aggregate interface.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\tagg := NewAggregate([]error{err1, err2})\n")
	g.Printf("//\tfmt.Println(agg)          // Output: [error1 message, error2 message]\n")
	g.Printf("//\tfmt.Printf(\"%%+v\\n\", agg) // Output: [error1 details, error2 details]\n")
	g.Printf("//\n")
	g.Printf("func NewAggregate(errlist []error) errors.Aggregate {\n")
	g.Printf("\treturn errors.NewAggregate(errlist)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// FilterOut wrapper
	g.Printf("// FilterOut removes all errors that match any of the matchers from the input error.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\tfiltered := FilterOut(agg, func(err error) bool {\n")
	g.Printf("//\t\treturn IsCode(err, ErrPageNotFound)\n")
	g.Printf("//\t})\n")
	g.Printf("//\tfmt.Println(filtered)\n")
	g.Printf("//\n")
	g.Printf("func FilterOut(err error, fns ...errors.Matcher) error {\n")
	g.Printf("\treturn errors.FilterOut(err, fns...)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// Flatten wrapper
	g.Printf("// Flatten takes an Aggregate and flattens them all into a single Aggregate, recursively.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\tflattened := Flatten(nestedAgg)\n")
	g.Printf("//\tfmt.Println(flattened)\n")
	g.Printf("//\n")
	g.Printf("func Flatten(agg errors.Aggregate) errors.Aggregate {\n")
	g.Printf("\treturn errors.Flatten(agg)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// Reduce wrapper
	g.Printf("// Reduce will return err or, if err is an Aggregate and only has one item, the first item.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\treduced := Reduce(agg)\n")
	g.Printf("//\tfmt.Println(reduced)\n")
	g.Printf("//\n")
	g.Printf("func Reduce(err error) error {\n")
	g.Printf("\treturn errors.Reduce(err)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// AggregateGoroutines wrapper
	g.Printf("// AggregateGoroutines runs the provided functions in parallel, stuffing all non-nil errors into the returned Aggregate.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\tagg := AggregateGoroutines(\n")
	g.Printf("//\t\tfunc() error { return doWork1() },\n")
	g.Printf("//\t\tfunc() error { return doWork2() },\n")
	g.Printf("//\t)\n")
	g.Printf("//\tif agg != nil {\n")
	g.Printf("//\t\tfmt.Println(agg)\n")
	g.Printf("//\t}\n")
	g.Printf("//\n")
	g.Printf("func AggregateGoroutines(funcs ...func() error) errors.Aggregate {\n")
	g.Printf("\treturn errors.AggregateGoroutines(funcs...)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// IsCode wrapper
	g.Printf("// IsCode reports whether any error in err's chain contains the given error code\n")
	g.Printf("// within the specified namespace.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\tif IsCode(err, ErrPageNotFound) {\n")
	g.Printf("//\t\tfmt.Println(\"Page not found\")\n")
	g.Printf("//\t}\n")
	g.Printf("//\n")
	g.Printf("func IsCode(err error, code int) bool {\n")
	g.Printf("\treturn errors.IsCode(err, Namespace, code)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// Is wrapper
	g.Printf("// Is reports whether any error in err's chain matches target.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\tif Is(err, io.EOF) {\n")
	g.Printf("//\t\tfmt.Println(\"End of stream\")\n")
	g.Printf("//\t}\n")
	g.Printf("//\n")
	g.Printf("func Is(err, target error) bool {\n")
	g.Printf("\treturn errors.Is(err, target)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// As wrapper
	g.Printf("// As finds the first error in err's chain that matches target, and if so, sets\n")
	g.Printf("// target to that error value and returns true.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\tvar target *CustomError\n")
	g.Printf("//\tif As(err, &target) {\n")
	g.Printf("//\t\tfmt.Println(target.Message)\n")
	g.Printf("//\t}\n")
	g.Printf("//\n")
	g.Printf("func As(err error, target interface{}) bool {\n")
	g.Printf("\treturn errors.As(err, target)\n")
	g.Printf("}\n")
	g.Printf("\n")

	// Unwrap wrapper
	g.Printf("// Unwrap returns the result of calling the Unwrap method on err, if err's\n")
	g.Printf("// type contains an Unwrap method returning error.\n")
	g.Printf("// Otherwise, Unwrap returns nil.\n")
	g.Printf("//\n")
	g.Printf("// Example:\n")
	g.Printf("//\n")
	g.Printf("//\tunwrapped := Unwrap(wrappedErr)\n")
	g.Printf("//\tfmt.Println(unwrapped)\n")
	g.Printf("//\n")
	g.Printf("func Unwrap(err error) error {\n")
	g.Printf("\treturn errors.Unwrap(err)\n")
	g.Printf("}\n")
}
func (g *Generator) generateDocs(typeName string) {
	if len(g.values) == 0 {
		log.Fatalf("no values defined for type %s", typeName)
	}

	tmpl, err := template.New("doc").Parse(errCodeDocPrefix)
	if err != nil {
		log.Fatalf("parsing doc template: %s", err)
	}
	var buf bytes.Buffer

	// Build the actual generate command with all parameters
	var cmdBuf strings.Builder
	cmdBuf.WriteString("codegen -type=int")
	if *projectFullName != "" {
		cmdBuf.WriteString(fmt.Sprintf(" -fullname=%s", *projectFullName))
	}
	if *namespace != "" {
		cmdBuf.WriteString(fmt.Sprintf(" -namespace=%s", *namespace))
	}
	if *projectSimpleName != *projectFullName {
		cmdBuf.WriteString(fmt.Sprintf(" -simplename=%s", *projectSimpleName))
	}
	if *docOutput != "" {
		cmdBuf.WriteString(fmt.Sprintf(" -doc-output=%s", *docOutput))
	}
	if *wrapper {
		cmdBuf.WriteString(" -wrapper")
	}
	if *wrapperOutput != "" && *wrapperOutput != "errors_generated.go" {
		cmdBuf.WriteString(fmt.Sprintf(" -wrapper-output=%s", *wrapperOutput))
	}
	generateCmd := cmdBuf.String()

	var tmplInfo = TemplateInfo{
		BackQuote:       "`",
		FullName:        *projectFullName,
		SimpleName:      *projectSimpleName,
		Namespace:       *namespace,
		GenerateCommand: generateCmd,
	}
	_ = tmpl.Execute(&buf, tmplInfo)

	// Count lines in template to determine starting line for error codes
	templateLineCount := bytes.Count(buf.Bytes(), []byte("\n"))

	// Generate code that will fail if the constants change value.
	g.Printf(buf.String())
	for i, v := range g.values {
		code, description := v.ParseComment()
		v.ParseReference()
		// Calculate line number in doc file (template lines + current index + 1 for 1-based indexing)
		// Template ends at line templateLineCount, first error code is at templateLineCount + 1
		v.docLine = templateLineCount + i + 1
		g.values[i] = v // Update the value with docLine

		// Determine what to display in documentation Reference column
		var refDisplay string
		if v.refMsg != "" && v.refURL != "" {
			// Has both Ref Msg and Ref URL: show "Ref Msg (see details: Ref URL)"
			refDisplay = fmt.Sprintf("%s (see details: %s)", v.refMsg, v.refURL)
		} else if v.refMsg != "" {
			// Has Ref Msg only: show Ref Msg
			refDisplay = v.refMsg
		} else if v.refURL != "" {
			// Has Ref URL only: show Ref URL
			refDisplay = v.refURL
		}
		// If neither, refDisplay remains empty

		g.Printf("| %s | %d | %s | %s | %s |\n", v.originalName, v.value, code, description, refDisplay)
	}
	g.Printf("\n")
}

// format returns the gofmt-ed contents of the Generator's buffer.
func (g *Generator) format() []byte {
	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		// Should never happen, but can arise when developing this code.
		// The user can compile the output to see the error.
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")

		return g.buf.Bytes()
	}

	return src
}

// Value represents a declared constant.
type Value struct {
	comment      string
	originalName string // The name of the constant.
	name         string // The name with trimmed prefix.
	// The value is stored as a bit pattern alone. The boolean tells us
	// whether to interpret it as an int64 or a uint64; the only place
	// this matters is when sorting.
	// Much of the time the str field is all we need; it is printed
	// by Value.String.
	value   uint64 // Will be converted to int64 when needed.
	signed  bool   // Whether the constant is a signed type.
	str     string // The string representation given by the "go/constant" package.
	line    int    // Line number in the source file.
	docLine int    // Line number in the generated doc file.
	refMsg  string // Reference message for documentation display.
	refURL  string // Reference URL for error code object.
	file    string // Source file name.
}

func (v *Value) String() string {
	return v.str
}

// ParseComment parse comment to http code and error code description.
func (v *Value) ParseComment() (string, string) {
	reg := regexp.MustCompile(`\w\s*-\s*(\d{3})\s*:\s*([A-Z].*)\s*\.\n*`)
	if !reg.MatchString(v.comment) {
		log.Printf("constant '%s' have wrong comment format, register with 500 as default", v.originalName)

		return "500", "Internal server error"
	}

	groups := reg.FindStringSubmatch(v.comment)
	if len(groups) != 3 {
		return "500", "Internal server error"
	}

	return groups[1], groups[2]
}

// ParseReference extracts reference URL and message from comment.
// Looks for "Reference Message: <msg>" or "Ref Msg: <msg>" pattern for documentation display.
// Looks for "Reference: <url>" or "Ref: <url>" pattern for error code object URL.
// If no Reference/Ref URL found, generates one based on the doc output file and git remote.
func (v *Value) ParseReference() string {
	// Parse reference message (for documentation display)
	refMsgReg := regexp.MustCompile(`(?i)(?:Reference\s+Message|Ref\s+Msg)\s*:\s*(.+)`)
	if matches := refMsgReg.FindStringSubmatch(v.comment); len(matches) > 1 {
		v.refMsg = strings.TrimSpace(matches[1])
	}

	// Parse reference URL (for error code object)
	refURLReg := regexp.MustCompile(`(?i)(?:Reference|Ref)\s*:\s*(https?://\S+)`)
	if matches := refURLReg.FindStringSubmatch(v.comment); len(matches) > 1 {
		v.refURL = matches[1]
	}

	// Determine which URL to use for error code object
	// If has Ref URL: use it for code
	// If no Ref URL: use auto-generated doc link
	if v.refURL != "" {
		return v.refURL
	}

	// No Ref URL: use auto-generated doc link
	if docPath != "" {
		return generateReferenceURL(v)
	}
	return ""
}

// generateReferenceURL generates a reference URL based on the doc output file and git remote.
func generateReferenceURL(v *Value) string {
	// Get git remote URL
	remoteURL := getGitRemoteURL()
	if remoteURL == "" {
		return ""
	}

	// Convert doc path to absolute path
	docFilePath := docPath
	if !filepath.IsAbs(docFilePath) {
		// Make it absolute relative to current directory
		absPath, err := filepath.Abs(docFilePath)
		if err != nil {
			return ""
		}
		docFilePath = absPath
	}

	// Get project root directory
	projectRoot := getProjectRoot()
	if projectRoot == "" {
		return ""
	}

	// Calculate relative path from project root
	relPath, err := filepath.Rel(projectRoot, docFilePath)
	if err != nil {
		return ""
	}

	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	// Generate URL: remote/blob/branch/path/to/file.md#L{line}
	// For now, use main as default branch
	branch := "main"

	// Clean remote URL - remove .git suffix if present
	repoURL := strings.TrimSuffix(remoteURL, ".git")

	// Use docLine if available, otherwise fall back to source line
	lineNum := v.docLine
	if lineNum == 0 {
		lineNum = v.line
	}

	// Generate the full URL
	return fmt.Sprintf("%s/blob/%s/%s#L%d", repoURL, branch, relPath, lineNum)
}

// getGitRemoteURL returns the git remote origin URL.
func getGitRemoteURL() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	url := strings.TrimSpace(string(output))

	// Convert SSH URL to HTTPS if needed
	// git@github.com:user/repo.git -> https://github.com/user/repo.git
	if strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			host := strings.TrimPrefix(parts[0], "git@")
			path := parts[1]
			url = fmt.Sprintf("https://%s/%s", host, path)
		}
	}

	return url
}

// getProjectRoot returns the project root directory (where .git is located).
func getProjectRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// nolint: gocognit
// genDecl processes one declaration clause.
func (f *File) genDecl(node ast.Node) bool {
	decl, ok := node.(*ast.GenDecl)
	if !ok || decl.Tok != token.CONST {
		// We only care about const declarations.
		return true
	}
	// The name of the type of the constants we are declaring.
	// Can change if this is a multi-element declaration.
	typ := ""
	// Loop over the elements of the declaration. Each element is a ValueSpec:
	// a list of names possibly followed by a type, possibly followed by values.
	// If the type and value are both missing, we carry down the type (and value,
	// but the "go/types" package takes care of that).
	for _, spec := range decl.Specs {
		vspec, _ := spec.(*ast.ValueSpec) // Guaranteed to succeed as this is CONST.
		if vspec.Type == nil && len(vspec.Values) > 0 {
			// "X = 1". With no type but a value. If the constant is untyped,
			// skip this vspec and reset the remembered type.
			typ = ""

			// If this is a simple type conversion, remember the type.
			// We don't mind if this is actually a call; a qualified call won't
			// be matched (that will be SelectorExpr, not Ident), and only unusual
			// situations will result in a function call that appears to be
			// a type conversion.
			ce, ok := vspec.Values[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			id, ok := ce.Fun.(*ast.Ident)
			if !ok {
				continue
			}
			typ = id.Name
		}
		if vspec.Type != nil {
			// "X T". We have a type. Remember it.
			ident, ok := vspec.Type.(*ast.Ident)
			if !ok {
				continue
			}
			typ = ident.Name
		}
		if typ != f.typeName {
			// This is not the type we're looking for.
			continue
		}
		// We now have a list of names (from one line of source code) all being
		// declared with the desired type.
		// Grab their names and actual values and store them in f.values.
		for _, name := range vspec.Names {
			if name.Name == "_" {
				continue
			}
			// This dance lets the type checker find the values for us. It's a
			// bit tricky: look up the object declared by the name, find its
			// types.Const, and extract its value.
			obj, ok := f.pkg.defs[name]
			if !ok {
				log.Fatalf("no value for constant %s", name)
			}
			info := obj.Type().Underlying().(*types.Basic).Info()
			if info&types.IsInteger == 0 {
				log.Fatalf("can't handle non-integer constant type %s", typ)
			}
			value := obj.(*types.Const).Val() // Guaranteed to succeed as this is CONST.
			if value.Kind() != constant.Int {
				log.Fatalf("can't happen: constant is not an integer %s", name)
			}
			i64, isInt := constant.Int64Val(value)
			u64, isUint := constant.Uint64Val(value)
			if !isInt && !isUint {
				log.Fatalf("internal error: value of %s is not an integer: %s", name, value.String())
			}
			if !isInt {
				u64 = uint64(i64)
			}
			// Get file and line information
			pos := f.pkg.fset.Position(name.Pos())
			fileName := pos.Filename
			lineNum := pos.Line

			v := Value{
				originalName: name.Name,
				value:        u64,
				signed:       info&types.IsUnsigned == 0,
				str:          value.String(),
				line:         lineNum,
				file:         fileName,
			}
			if vspec.Doc != nil && vspec.Doc.Text() != "" {
				v.comment = vspec.Doc.Text()
			} else if c := vspec.Comment; c != nil && len(c.List) == 1 {
				v.comment = c.Text()
			}

			v.name = strings.TrimPrefix(v.originalName, f.trimPrefix)
			f.values = append(f.values, v)
		}
	}

	return false
}
