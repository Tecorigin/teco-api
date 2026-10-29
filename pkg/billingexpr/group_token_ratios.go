package billingexpr

import (
	"fmt"
	"math"
	"strings"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

func ApplyGroupTokenRatios(expression string, inputRatio, outputRatio float64) (string, error) {
	if inputRatio < 0 || outputRatio < 0 || math.IsNaN(inputRatio) || math.IsNaN(outputRatio) || math.IsInf(inputRatio, 0) || math.IsInf(outputRatio, 0) {
		return "", fmt.Errorf("group token coefficients must be finite and non-negative")
	}
	if inputRatio == 1 && outputRatio == 1 {
		return expression, nil
	}
	_, body := ParseExprVersion(expression)
	tree, err := parser.Parse(body)
	if err != nil {
		return "", err
	}
	adjusted, err := groupTokenPriceExpression(tree.Node, inputRatio, outputRatio)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(expression, "v1:") {
		adjusted = "v1:" + adjusted
	}
	return adjusted, nil
}

func groupTokenPriceDimensions(node ast.Node) (input, output, media bool) {
	ast.Find(node, func(candidate ast.Node) bool {
		identifier, ok := candidate.(*ast.IdentifierNode)
		if !ok {
			return false
		}
		switch identifier.Value {
		case "p", "cr", "cc", "cc1h":
			input = true
		case "c":
			output = true
		case "img", "img_o", "ai", "ao", "asr_s", "img_n", "vid_s", "tts_c":
			media = true
		}
		return false
	})
	return
}

func groupTokenPriceExpression(node ast.Node, inputRatio, outputRatio float64) (string, error) {
	switch expression := node.(type) {
	case *ast.ConditionalNode:
		matched, err := groupTokenPriceExpression(expression.Exp1, inputRatio, outputRatio)
		if err != nil {
			return "", err
		}
		fallback, err := groupTokenPriceExpression(expression.Exp2, inputRatio, outputRatio)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s ? %s : %s)", expression.Cond.String(), matched, fallback), nil
	case *ast.CallNode:
		if function, ok := expression.Callee.(*ast.IdentifierNode); ok && function.Value == "tier" && len(expression.Arguments) == 2 {
			price, err := groupTokenPriceExpression(expression.Arguments[1], inputRatio, outputRatio)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("tier(%s, %s)", expression.Arguments[0].String(), price), nil
		}
	case *ast.BinaryNode:
		if expression.Operator == "+" || expression.Operator == "-" {
			left, err := groupTokenPriceExpression(expression.Left, inputRatio, outputRatio)
			if err != nil {
				return "", err
			}
			right, err := groupTokenPriceExpression(expression.Right, inputRatio, outputRatio)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("(%s %s %s)", left, expression.Operator, right), nil
		}
	}

	input, output, media := groupTokenPriceDimensions(node)
	if input && !output && !media {
		return fmt.Sprintf("(%s * %g)", node.String(), inputRatio), nil
	}
	if output && !input && !media {
		return fmt.Sprintf("(%s * %g)", node.String(), outputRatio), nil
	}
	if !input && !output {
		return node.String(), nil
	}
	if expression, ok := node.(*ast.BinaryNode); ok && (expression.Operator == "*" || expression.Operator == "/") {
		leftInput, leftOutput, leftMedia := groupTokenPriceDimensions(expression.Left)
		rightInput, rightOutput, rightMedia := groupTokenPriceDimensions(expression.Right)
		if !rightInput && !rightOutput && !rightMedia {
			left, err := groupTokenPriceExpression(expression.Left, inputRatio, outputRatio)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("(%s %s %s)", left, expression.Operator, expression.Right.String()), nil
		}
		if expression.Operator == "*" && !leftInput && !leftOutput && !leftMedia {
			right, err := groupTokenPriceExpression(expression.Right, inputRatio, outputRatio)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("(%s * %s)", expression.Left.String(), right), nil
		}
	}
	return "", fmt.Errorf("cannot apply group token coefficients to mixed pricing term %s; separate text input, text output and media prices", node.String())
}
