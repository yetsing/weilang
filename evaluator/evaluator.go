package evaluator

import (
	"fmt"
	"weilang/ast"
	"weilang/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.TypeIs(object.ERROR_OBJ)
	}
	return false
}

func Eval(node ast.Node) object.Object {
	switch node := node.(type) {

	// 语句
	case *ast.Program:
		return evalProgram(node)

	case *ast.BlockStatement:
		return evalBlockStatements(node)

	case *ast.IfStatement:
		return evalIfStatement(node)

	case *ast.ExpressionStatement:
		return Eval(node.Expression)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	// 表达式
	case *ast.UnaryExpression:
		operand := Eval(node.Operand)
		if isError(operand) {
			return operand
		}
		return evalUnaryExpression(node.Operator, operand)

	case *ast.BinaryOpExpression:
		left := Eval(node.Left)
		if isError(left) {
			return left
		}
		right := Eval(node.Right)
		if isError(right) {
			return right
		}
		return evalBinaryOpExpression(node.Operator, left, right)

	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.NullLiteral:
		return NULL
	}

	return nil
}

func evalProgram(program *ast.Program) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}
	return result
}

func evalBlockStatements(block *ast.BlockStatement) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = Eval(statement)

		if result != nil {
			if result.TypeIs(object.RETURN_VALUE_OBJ) || result.TypeIs(object.ERROR_OBJ) {
				return result
			}
		}
	}

	return result
}

func evalIfStatement(is *ast.IfStatement) object.Object {
	for _, branch := range is.IfBranches {
		condition := Eval(branch.Condition)
		if isError(condition) {
			return condition
		}
		if isTruthy(condition) {
			return Eval(branch.Body)
		}
	}

	if is.ElseBody != nil {
		return Eval(is.ElseBody)
	} else {
		return NULL
	}
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case TRUE:
		return true
	case FALSE:
		return false
	case NULL:
		return false
	default:
		switch obj := obj.(type) {
		case *object.Integer:
			return obj.Value != 0
		}
		return true
	}
}

func evalUnaryExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "not":
		return evalNotOperatorExpression(right)
	case "-":
		return evalMinusUnaryOperatorExpression(right)
	case "+":
		return evalPlusUnaryOperatorExpression(right)
	case "~":
		return evalBitwiseNotOperatorExpression(right)
	default:
		return object.NewError("unsupported operand type for %s: '%s'", operator, right.Type())
	}
}

func evalNotOperatorExpression(operand object.Object) object.Object {
	return nativeBoolToBooleanObject(!isTruthy(operand))
}

func evalMinusUnaryOperatorExpression(right object.Object) object.Object {
	if right.TypeNotIs(object.INTEGER_OBJ) {
		message := fmt.Sprintf("unsupported operand type for -: '%s'", right.Type())
		return &object.Error{Message: message}
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

func evalPlusUnaryOperatorExpression(right object.Object) object.Object {
	if right.TypeNotIs(object.INTEGER_OBJ) {
		message := fmt.Sprintf("unsupported operand type for +: '%s'", right.Type())
		return &object.Error{Message: message}
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: value}
}

func evalBitwiseNotOperatorExpression(right object.Object) object.Object {
	if right.TypeNotIs(object.INTEGER_OBJ) {
		message := fmt.Sprintf("bad operand type for unary +: '%s'", right.Type())
		return &object.Error{Message: message}
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: ^value}
}

func evalBinaryOpExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerBinaryOpExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	default:
		msg := fmt.Sprintf("unsupported operand type for %s: '%s' and '%s'",
			operator, left.Type(), right.Type())
		return &object.Error{Message: msg}
	}
}

func evalIntegerBinaryOpExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "%":
		return &object.Integer{Value: leftVal % rightVal}
	case ">>":
		return &object.Integer{Value: leftVal >> rightVal}
	case "<<":
		return &object.Integer{Value: leftVal << rightVal}
	case "&":
		return &object.Integer{Value: leftVal & rightVal}
	case "^":
		return &object.Integer{Value: leftVal ^ rightVal}
	case "|":
		return &object.Integer{Value: leftVal | rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		msg := fmt.Sprintf("unsupported operand type for %s: '%s' and '%s'",
			operator, left.Type(), right.Type())
		return &object.Error{Message: msg}
	}
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}
