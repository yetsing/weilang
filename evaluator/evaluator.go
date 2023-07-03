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

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {

	// 语句
	case *ast.Program:
		return evalProgram(node, env)

	case *ast.BlockStatement:
		return evalBlockStatements(node, env)

	case *ast.VarStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)

	case *ast.IfStatement:
		return evalIfStatement(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	// 表达式
	case *ast.UnaryExpression:
		operand := Eval(node.Operand, env)
		if isError(operand) {
			return operand
		}
		return evalUnaryExpression(node.Operator, operand)

	case *ast.BinaryOpExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalBinaryOpExpression(node.Operator, left, right)

	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(function, args)

	case *ast.FunctionLiteral:
		return object.NewFunction(node, env)

	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.NullLiteral:
		return NULL
	}

	return nil
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}
	return result
}

func evalBlockStatements(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if result != nil {
			if result.TypeIs(object.RETURN_VALUE_OBJ) || result.TypeIs(object.ERROR_OBJ) {
				return result
			}
		}
	}

	return result
}

func evalIfStatement(is *ast.IfStatement, env *object.Environment) object.Object {
	for _, branch := range is.IfBranches {
		condition := Eval(branch.Condition, env)
		if isError(condition) {
			return condition
		}
		if isTruthy(condition) {
			return Eval(branch.Body, env)
		}
	}

	if is.ElseBody != nil {
		return Eval(is.ElseBody, env)
	} else {
		return NULL
	}
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	function, ok := fn.(*object.Function)
	if !ok {
		return object.NewError("not a function: '%s'", fn.Type())
	}

	if len(args) != len(function.Parameters) {
		return object.NewError("function expected %d arguments but got %d", len(function.Parameters), len(args))
	}

	extendedEnv := extendFunctionEnv(function, args)
	evaluated := Eval(function.Body, extendedEnv)
	if returnValue, ok := evaluated.(*object.ReturnValue); ok {
		return returnValue.Value
	}
	return NULL
}

func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for paramIdx, parameter := range fn.Parameters {
		env.Set(parameter.Value, args[paramIdx])
	}

	return env
}

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, exp := range exps {
		evaluated := Eval(exp, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
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
	case left.TypeIs(object.INTEGER_OBJ) && right.TypeIs(object.INTEGER_OBJ):
		return evalIntegerBinaryOpExpression(operator, left, right)
	case left.TypeIs(object.STRING_OBJ) && right.TypeIs(object.STRING_OBJ):
		return evalStringBinaryOpExpression(operator, left, right)

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

func evalStringBinaryOpExpression(
	operator string,
	left, right object.Object,
) object.Object {
	if operator != "+" {
		return object.NewError("unsupported operand type for %s: 'str' and 'str'", operator)
	}

	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value
	return &object.String{Value: leftVal + rightVal}
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	val, ok := env.Get(node.Value)
	if !ok {
		return object.NewError("identifier not found: '%s'", node.Value)
	}
	return val
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}
