package mutation

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"github.com/aiden/mutcheck/internal/domain"
)

// Discover returns supported, single-node mutation candidates in AST traversal order.
func Discover(sourcePath string) ([]domain.Mutant, error) {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, sourcePath, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse source: %w", err)
	}

	mutants := make([]domain.Mutant, 0)
	index := 0
	ast.Inspect(file, func(node ast.Node) bool {
		original, replacement, operator, ok := mutationFor(node)
		if !ok {
			return true
		}
		index++
		position := fileSet.Position(node.Pos())
		mutants = append(mutants, domain.Mutant{
			ID:             fmt.Sprintf("M%03d", index),
			CandidateIndex: index,
			SourcePath:     filepath.ToSlash(filepath.Base(sourcePath)),
			Line:           position.Line,
			Column:         position.Column,
			Operator:       operator,
			Original:       original,
			Replacement:    replacement,
		})
		return true
	})
	return mutants, nil
}

// Apply recreates the source AST, changes the selected candidate, and formats valid Go source.
func Apply(sourcePath string, candidateIndex int) ([]byte, error) {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, sourcePath, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse source: %w", err)
	}

	seen, changed := 0, false
	ast.Inspect(file, func(node ast.Node) bool {
		if changed {
			return false
		}
		if !isMutable(node) {
			return true
		}
		seen++
		if seen == candidateIndex {
			changed = applyMutation(node)
		}
		return true
	})
	if !changed {
		return nil, fmt.Errorf("mutation candidate %d was not found", candidateIndex)
	}

	var output bytes.Buffer
	if err := format.Node(&output, fileSet, file); err != nil {
		return nil, fmt.Errorf("format mutated source: %w", err)
	}
	return output.Bytes(), nil
}

func isMutable(node ast.Node) bool {
	_, _, _, ok := mutationFor(node)
	return ok
}

func mutationFor(node ast.Node) (original, replacement, operator string, ok bool) {
	switch expression := node.(type) {
	case *ast.BinaryExpr:
		if replacementToken, supported := replacementForToken(expression.Op); supported {
			return expression.Op.String(), replacementToken.String(), "binary", true
		}
	case *ast.Ident:
		if expression.Name == "true" {
			return "true", "false", "boolean", true
		}
		if expression.Name == "false" {
			return "false", "true", "boolean", true
		}
	}
	return "", "", "", false
}

func applyMutation(node ast.Node) bool {
	switch expression := node.(type) {
	case *ast.BinaryExpr:
		replacement, ok := replacementForToken(expression.Op)
		if !ok {
			return false
		}
		expression.Op = replacement
		return true
	case *ast.Ident:
		switch expression.Name {
		case "true":
			expression.Name = "false"
			return true
		case "false":
			expression.Name = "true"
			return true
		}
	}
	return false
}

func replacementForToken(operator token.Token) (token.Token, bool) {
	replacements := map[token.Token]token.Token{
		token.GTR: token.GEQ, token.GEQ: token.GTR,
		token.LSS: token.LEQ, token.LEQ: token.LSS,
		token.EQL: token.NEQ, token.NEQ: token.EQL,
		token.ADD: token.SUB, token.SUB: token.ADD,
		token.LAND: token.LOR, token.LOR: token.LAND,
	}
	replacement, ok := replacements[operator]
	return replacement, ok
}
