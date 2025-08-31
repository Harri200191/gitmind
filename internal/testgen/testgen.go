package testgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/Harri200191/gitmind/internal/config"
)

// TestGenerator handles automatic test generation
type TestGenerator struct {
	config config.Config
}

// FunctionInfo represents information about a function
type FunctionInfo struct {
	Name       string                 `json:"name"`
	Package    string                 `json:"package"`
	File       string                 `json:"file"`
	Parameters []Parameter            `json:"parameters"`
	Returns    []ReturnValue          `json:"returns"`
	Comments   string                 `json:"comments"`
	IsExported bool                   `json:"is_exported"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ReturnValue represents a return value
type ReturnValue struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// TestCase represents a generated test case
type TestCase struct {
	Function    string   `json:"function"`
	TestName    string   `json:"test_name"`
	Setup       []string `json:"setup"`
	Inputs      []string `json:"inputs"`
	Expected    []string `json:"expected"`
	Assertions  []string `json:"assertions"`
	Description string   `json:"description"`
}

// TestFile represents a complete test file
type TestFile struct {
	Package   string     `json:"package"`
	Imports   []string   `json:"imports"`
	TestCases []TestCase `json:"test_cases"`
	Content   string     `json:"content"`
}

// New creates a new test generator
func New(cfg config.Config) *TestGenerator {
	return &TestGenerator{config: cfg}
}

// AnalyzeChangedFunctions extracts function information from git diff
func (tg *TestGenerator) AnalyzeChangedFunctions(diff string) ([]FunctionInfo, error) {
	if !tg.config.TestGeneration.Enabled {
		return nil, nil
	}

	// Parse the diff to get changed files
	changedFiles := tg.extractChangedGoFiles(diff)

	var functions []FunctionInfo

	for _, file := range changedFiles {
		fileFunctions, err := tg.analyzeFunctionsInFile(file, diff)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Error analyzing file %s: %v\n", file, err)
			continue
		}
		functions = append(functions, fileFunctions...)
	}

	return functions, nil
}

// GenerateTests creates test cases for the given functions
func (tg *TestGenerator) GenerateTests(functions []FunctionInfo) (map[string]TestFile, error) {
	if !tg.config.TestGeneration.Enabled {
		return nil, nil
	}

	testFiles := make(map[string]TestFile)

	// Group functions by package
	packageFunctions := tg.groupFunctionsByPackage(functions)

	for pkg, pkgFunctions := range packageFunctions {
		testFile, err := tg.generateTestFile(pkg, pkgFunctions)
		if err != nil {
			return nil, fmt.Errorf("error generating tests for package %s: %v", pkg, err)
		}

		testFiles[pkg] = testFile
	}

	return testFiles, nil
}

// WriteTestFiles writes the generated test files to disk
func (tg *TestGenerator) WriteTestFiles(testFiles map[string]TestFile) error {
	for pkg, testFile := range testFiles {
		outputPath := tg.getTestFilePath(pkg)

		// Ensure directory exists
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directory %s: %v", dir, err)
		}

		// Write test file
		if err := os.WriteFile(outputPath, []byte(testFile.Content), 0644); err != nil {
			return fmt.Errorf("error writing test file %s: %v", outputPath, err)
		}

		fmt.Printf("Generated test file: %s\n", outputPath)

		// Auto-stage if configured
		if tg.config.TestGeneration.AutoStage {
			// This would need git integration
			fmt.Printf("Auto-staging test file: %s\n", outputPath)
		}
	}

	return nil
}

// extractChangedGoFiles gets Go files from the diff
func (tg *TestGenerator) extractChangedGoFiles(diff string) []string {
	var files []string
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			file := strings.TrimPrefix(line, "+++ b/")
			if strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go") {
				files = append(files, file)
			}
		}
	}

	return files
}

// analyzeFunctionsInFile parses a Go file and extracts function information
func (tg *TestGenerator) analyzeFunctionsInFile(filename, diff string) ([]FunctionInfo, error) {
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Parse the file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var functions []FunctionInfo

	// Extract changed line numbers from diff
	changedLines := tg.extractChangedLines(filename, diff)

	// Walk the AST to find functions
	ast.Inspect(node, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Name.IsExported() || tg.shouldIncludePrivateFunction(fn.Name.Name) {
				pos := fset.Position(fn.Pos())

				// Check if this function was modified
				if tg.isFunctionChanged(pos.Line, changedLines) {
					funcInfo := tg.extractFunctionInfo(fn, node.Name.Name, filename, fset)
					functions = append(functions, funcInfo)
				}
			}
		}
		return true
	})

	return functions, nil
}

// extractFunctionInfo creates FunctionInfo from AST node
func (tg *TestGenerator) extractFunctionInfo(fn *ast.FuncDecl, packageName, filename string, fset *token.FileSet) FunctionInfo {
	funcInfo := FunctionInfo{
		Name:       fn.Name.Name,
		Package:    packageName,
		File:       filename,
		IsExported: fn.Name.IsExported(),
		Metadata:   make(map[string]interface{}),
	}

	// Extract parameters
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			paramType := tg.typeToString(field.Type)
			for _, name := range field.Names {
				funcInfo.Parameters = append(funcInfo.Parameters, Parameter{
					Name: name.Name,
					Type: paramType,
				})
			}
		}
	}

	// Extract return values
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			returnType := tg.typeToString(field.Type)
			if len(field.Names) > 0 {
				for _, name := range field.Names {
					funcInfo.Returns = append(funcInfo.Returns, ReturnValue{
						Name: name.Name,
						Type: returnType,
					})
				}
			} else {
				funcInfo.Returns = append(funcInfo.Returns, ReturnValue{
					Type: returnType,
				})
			}
		}
	}

	// Extract comments
	if fn.Doc != nil {
		funcInfo.Comments = fn.Doc.Text()
	}

	return funcInfo
}

// generateTestFile creates a complete test file for a package
func (tg *TestGenerator) generateTestFile(pkg string, functions []FunctionInfo) (TestFile, error) {
	testFile := TestFile{
		Package: pkg + "_test",
		Imports: []string{
			"testing",
		},
	}

	// Add package import if testing external package
	if pkg != "main" {
		testFile.Imports = append(testFile.Imports, fmt.Sprintf("\"%s\"", pkg))
	}

	// Generate test cases for each function
	for _, fn := range functions {
		testCases := tg.generateTestCases(fn)
		testFile.TestCases = append(testFile.TestCases, testCases...)
	}

	// Generate the file content
	content := tg.buildTestFileContent(testFile)
	testFile.Content = content

	return testFile, nil
}

// generateTestCases creates test cases for a function
func (tg *TestGenerator) generateTestCases(fn FunctionInfo) []TestCase {
	var testCases []TestCase

	// Generate basic test case
	basicTest := TestCase{
		Function:    fn.Name,
		TestName:    fmt.Sprintf("Test%s", fn.Name),
		Description: fmt.Sprintf("Test basic functionality of %s", fn.Name),
	}

	// Generate setup code
	basicTest.Setup = tg.generateSetup(fn)

	// Generate input values
	basicTest.Inputs = tg.generateInputs(fn)

	// Generate expected values
	basicTest.Expected = tg.generateExpected(fn)

	// Generate assertions
	basicTest.Assertions = tg.generateAssertions(fn)

	testCases = append(testCases, basicTest)

	// Generate edge case tests if applicable
	if tg.shouldGenerateEdgeCases(fn) {
		edgeTest := TestCase{
			Function:    fn.Name,
			TestName:    fmt.Sprintf("Test%s_EdgeCases", fn.Name),
			Description: fmt.Sprintf("Test edge cases of %s", fn.Name),
		}

		edgeTest.Setup = tg.generateEdgeCaseSetup(fn)
		edgeTest.Inputs = tg.generateEdgeCaseInputs(fn)
		edgeTest.Expected = tg.generateEdgeCaseExpected(fn)
		edgeTest.Assertions = tg.generateAssertions(fn)

		testCases = append(testCases, edgeTest)
	}

	// Generate error case tests
	if tg.hasErrorReturn(fn) {
		errorTest := TestCase{
			Function:    fn.Name,
			TestName:    fmt.Sprintf("Test%s_Error", fn.Name),
			Description: fmt.Sprintf("Test error handling of %s", fn.Name),
		}

		errorTest.Setup = tg.generateErrorSetup(fn)
		errorTest.Inputs = tg.generateErrorInputs(fn)
		errorTest.Expected = tg.generateErrorExpected(fn)
		errorTest.Assertions = tg.generateErrorAssertions(fn)

		testCases = append(testCases, errorTest)
	}

	return testCases
}

// Helper methods
func (tg *TestGenerator) extractChangedLines(filename, diff string) map[int]bool {
	changedLines := make(map[int]bool)
	lines := strings.Split(diff, "\n")

	inFile := false
	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			file := strings.TrimPrefix(line, "+++ b/")
			inFile = (file == filename)
			continue
		}

		if !inFile {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			// Parse hunk header for line numbers
			// This is a simplified version
			continue
		}

		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			// Mark as changed (simplified)
			changedLines[len(changedLines)] = true
		}
	}

	return changedLines
}

func (tg *TestGenerator) isFunctionChanged(line int, changedLines map[int]bool) bool {
	// Simplified check - in real implementation, would need better line tracking
	return len(changedLines) > 0
}

func (tg *TestGenerator) shouldIncludePrivateFunction(name string) bool {
	// Include private functions for comprehensive testing
	return true
}

func (tg *TestGenerator) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return tg.typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + tg.typeToString(t.X)
	case *ast.ArrayType:
		return "[]" + tg.typeToString(t.Elt)
	default:
		return "interface{}"
	}
}

func (tg *TestGenerator) groupFunctionsByPackage(functions []FunctionInfo) map[string][]FunctionInfo {
	groups := make(map[string][]FunctionInfo)
	for _, fn := range functions {
		groups[fn.Package] = append(groups[fn.Package], fn)
	}
	return groups
}

func (tg *TestGenerator) getTestFilePath(pkg string) string {
	outputDir := tg.config.TestGeneration.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	return filepath.Join(outputDir, pkg+"_test.go")
}

func (tg *TestGenerator) generateSetup(fn FunctionInfo) []string {
	var setup []string

	// Basic setup based on parameters
	for _, param := range fn.Parameters {
		switch param.Type {
		case "string":
			setup = append(setup, fmt.Sprintf("%s := \"test\"", param.Name))
		case "int", "int64", "int32":
			setup = append(setup, fmt.Sprintf("%s := 42", param.Name))
		case "bool":
			setup = append(setup, fmt.Sprintf("%s := true", param.Name))
		default:
			setup = append(setup, fmt.Sprintf("// TODO: setup %s of type %s", param.Name, param.Type))
		}
	}

	return setup
}

func (tg *TestGenerator) generateInputs(fn FunctionInfo) []string {
	var inputs []string
	for _, param := range fn.Parameters {
		inputs = append(inputs, param.Name)
	}
	return inputs
}

func (tg *TestGenerator) generateExpected(fn FunctionInfo) []string {
	var expected []string

	for i, ret := range fn.Returns {
		switch ret.Type {
		case "string":
			expected = append(expected, fmt.Sprintf("expected%d := \"expected\"", i))
		case "int", "int64", "int32":
			expected = append(expected, fmt.Sprintf("expected%d := 42", i))
		case "bool":
			expected = append(expected, fmt.Sprintf("expected%d := true", i))
		case "error":
			expected = append(expected, fmt.Sprintf("expected%d := error(nil)", i))
		default:
			expected = append(expected, fmt.Sprintf("// TODO: define expected%d of type %s", i, ret.Type))
		}
	}

	return expected
}

func (tg *TestGenerator) generateAssertions(fn FunctionInfo) []string {
	var assertions []string

	// Generate function call
	var callArgs []string
	for _, param := range fn.Parameters {
		callArgs = append(callArgs, param.Name)
	}

	var resultVars []string
	for i := range fn.Returns {
		resultVars = append(resultVars, fmt.Sprintf("result%d", i))
	}

	if len(resultVars) > 0 {
		call := fmt.Sprintf("%s := %s(%s)",
			strings.Join(resultVars, ", "),
			fn.Name,
			strings.Join(callArgs, ", "))
		assertions = append(assertions, call)

		// Generate assertions for each return value
		for i, ret := range fn.Returns {
			if ret.Type == "error" {
				assertions = append(assertions, fmt.Sprintf("if result%d != nil {", i))
				assertions = append(assertions, fmt.Sprintf("\tt.Errorf(\"Unexpected error: %%v\", result%d)", i))
				assertions = append(assertions, "}")
			} else {
				assertions = append(assertions, fmt.Sprintf("if result%d != expected%d {", i, i))
				assertions = append(assertions, fmt.Sprintf("\tt.Errorf(\"Expected %%v, got %%v\", expected%d, result%d)", i, i))
				assertions = append(assertions, "}")
			}
		}
	}

	return assertions
}

func (tg *TestGenerator) shouldGenerateEdgeCases(fn FunctionInfo) bool {
	// Generate edge cases for functions with numeric or string parameters
	for _, param := range fn.Parameters {
		if strings.Contains(param.Type, "int") || param.Type == "string" {
			return true
		}
	}
	return false
}

func (tg *TestGenerator) hasErrorReturn(fn FunctionInfo) bool {
	for _, ret := range fn.Returns {
		if ret.Type == "error" {
			return true
		}
	}
	return false
}

func (tg *TestGenerator) generateEdgeCaseSetup(fn FunctionInfo) []string {
	var setup []string

	for _, param := range fn.Parameters {
		switch param.Type {
		case "string":
			setup = append(setup, fmt.Sprintf("%s := \"\"", param.Name)) // empty string
		case "int", "int64", "int32":
			setup = append(setup, fmt.Sprintf("%s := 0", param.Name)) // zero value
		default:
			setup = append(setup, fmt.Sprintf("// TODO: edge case setup for %s", param.Name))
		}
	}

	return setup
}

func (tg *TestGenerator) generateEdgeCaseInputs(fn FunctionInfo) []string {
	return tg.generateInputs(fn) // Same as regular inputs for now
}

func (tg *TestGenerator) generateEdgeCaseExpected(fn FunctionInfo) []string {
	return tg.generateExpected(fn) // Same as regular expected for now
}

func (tg *TestGenerator) generateErrorSetup(fn FunctionInfo) []string {
	var setup []string

	for _, param := range fn.Parameters {
		switch param.Type {
		case "string":
			setup = append(setup, fmt.Sprintf("%s := \"invalid\"", param.Name))
		default:
			setup = append(setup, fmt.Sprintf("// TODO: error case setup for %s", param.Name))
		}
	}

	return setup
}

func (tg *TestGenerator) generateErrorInputs(fn FunctionInfo) []string {
	return tg.generateInputs(fn)
}

func (tg *TestGenerator) generateErrorExpected(fn FunctionInfo) []string {
	var expected []string

	for i, ret := range fn.Returns {
		if ret.Type == "error" {
			expected = append(expected, fmt.Sprintf("expectedErr%d := \"some error\"", i))
		} else {
			expected = append(expected, fmt.Sprintf("// TODO: define expected%d for error case", i))
		}
	}

	return expected
}

func (tg *TestGenerator) generateErrorAssertions(fn FunctionInfo) []string {
	var assertions []string

	// Similar to regular assertions but expecting errors
	var callArgs []string
	for _, param := range fn.Parameters {
		callArgs = append(callArgs, param.Name)
	}

	var resultVars []string
	for i := range fn.Returns {
		resultVars = append(resultVars, fmt.Sprintf("result%d", i))
	}

	if len(resultVars) > 0 {
		call := fmt.Sprintf("%s := %s(%s)",
			strings.Join(resultVars, ", "),
			fn.Name,
			strings.Join(callArgs, ", "))
		assertions = append(assertions, call)

		// Check for expected errors
		for i, ret := range fn.Returns {
			if ret.Type == "error" {
				assertions = append(assertions, fmt.Sprintf("if result%d == nil {", i))
				assertions = append(assertions, "\tt.Error(\"Expected error but got nil\")")
				assertions = append(assertions, "}")
			}
		}
	}

	return assertions
}

func (tg *TestGenerator) buildTestFileContent(testFile TestFile) string {
	var content strings.Builder

	// Package declaration
	content.WriteString(fmt.Sprintf("package %s\n\n", testFile.Package))

	// Imports
	content.WriteString("import (\n")
	for _, imp := range testFile.Imports {
		content.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}
	content.WriteString(")\n\n")

	// Test functions
	for _, testCase := range testFile.TestCases {
		content.WriteString(tg.buildTestFunction(testCase))
		content.WriteString("\n")
	}

	return content.String()
}

func (tg *TestGenerator) buildTestFunction(testCase TestCase) string {
	var content strings.Builder

	// Function signature
	content.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testCase.TestName))

	if testCase.Description != "" {
		content.WriteString(fmt.Sprintf("\t// %s\n", testCase.Description))
	}

	// Setup
	for _, setup := range testCase.Setup {
		content.WriteString(fmt.Sprintf("\t%s\n", setup))
	}

	if len(testCase.Setup) > 0 {
		content.WriteString("\n")
	}

	// Expected values
	for _, expected := range testCase.Expected {
		content.WriteString(fmt.Sprintf("\t%s\n", expected))
	}

	if len(testCase.Expected) > 0 {
		content.WriteString("\n")
	}

	// Assertions
	for _, assertion := range testCase.Assertions {
		content.WriteString(fmt.Sprintf("\t%s\n", assertion))
	}

	content.WriteString("}\n")

	return content.String()
}
