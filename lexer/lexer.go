package lexer

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
	"weilang/token"
)

type Lexer struct {
	// Filename 文件名
	Filename string
	input    string
	// current index of ch in ucodes
	index int
	// current char
	ch rune
	// unicode 列表
	ucodes   []rune
	position token.Position
	// 标记索引和位置，方便计算 Token 的 start end
	markIndex    int
	markPosition token.Position
}

func stringFromFilename(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf(`read file "%s" error: %v\n`, filename, err)
		os.Exit(1)
	}
	return string(data)
}

func New(input string) *Lexer {
	l := &Lexer{input: input, index: -1}
	l.init()
	l.readChar()
	return l
}

func NewWithFilename(filename string) *Lexer {
	input := stringFromFilename(filename)
	l := New(input)
	l.Filename = filename
	return l
}

func (l *Lexer) NextToken() token.Token {
	var ttype token.TokenType

	l.skipWhitespace()
	l.mark()

	switch l.ch {
	case '=':
		if l.peekCharIs('=') {
			l.readChar()
			ttype = token.EQ
		} else {
			ttype = token.ASSIGN
		}
	case '+':
		ttype = token.PLUS
	case '-':
		ttype = token.MINUS
	case '!':
		if l.peekCharIs('=') {
			l.readChar()
			ttype = token.NOT_EQ
		} else {
			l.readChar()
			tok := l.buildToken(token.ILLEGAL)
			tok.Literal = "invalid char !"
			return tok
		}
	case '/':
		if l.peekCharIs('/') {
			return l.readComment()
		} else {
			ttype = token.SLASH
		}
	case '*':
		ttype = token.ASTERISK
	case '%':
		ttype = token.MODULO
	case '<':
		if l.peekCharIs('=') {
			l.readChar()
			ttype = token.LESS_EQUAL_THAN
		} else if l.peekCharIs('<') {
			l.readChar()
			ttype = token.LEFT_SHIFT
		} else {
			ttype = token.LESS_THAN
		}
	case '>':
		if l.peekCharIs('=') {
			l.readChar()
			ttype = token.GREAT_EQUAL_THAN
		} else if l.peekCharIs('>') {
			l.readChar()
			ttype = token.RIGHT_SHIFT
		} else {
			ttype = token.GREAT_THAN
		}
	case '~':
		ttype = token.BITWISE_NOT
	case '&':
		ttype = token.BITWISE_AND
	case '^':
		ttype = token.BITWISE_XOR
	case '|':
		ttype = token.BITWISE_OR
	case ';':
		ttype = token.SEMICOLON
	case ':':
		ttype = token.COLON
	case ',':
		ttype = token.COMMA
	case '{':
		ttype = token.LBRACE
	case '}':
		ttype = token.RBRACE
	case '(':
		ttype = token.LPAREN
	case ')':
		ttype = token.RPAREN
	case '"':
		return l.readString(l.ch)
	case '\'':
		return l.readString(l.ch)
	case '`':
		return l.readRawString()
	case '[':
		ttype = token.LBRACKET
	case ']':
		ttype = token.RBRACKET
	case '.':
		ttype = token.DOT
	case 0:
		ttype = token.EOF
	default:
		if isIdentifierStart(l.ch) {
			return l.readIdentifier()
		} else if isDigit(l.ch) {
			return l.readNumber()
		} else {
			ttype = token.ILLEGAL
			ch := l.ch
			l.readChar()
			tok := l.buildToken(token.ILLEGAL)
			tok.Literal = "invalid char " + string(ch)
			return tok
		}
	}

	l.readChar()
	return l.buildToken(ttype)
}

func (l *Lexer) GetLines() []string {
	return strings.Split(l.input, "\n")
}

func (l *Lexer) init() {
	l.ucodes = []rune(l.input)
	l.position.Line = 0
	l.position.Column = -1
	l.Filename = "<input>"
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readChar() {
	l.index++
	if l.index >= len(l.ucodes) {
		l.ch = 0
	} else {
		if l.ch == '\n' {
			l.position.Line++
			l.position.Column = -1
		}
		l.ch = l.ucodes[l.index]
		l.position.Column++
	}
}

func (l *Lexer) advance(n int) {
	for i := 0; i < n; i++ {
		l.readChar()
	}
}

func (l *Lexer) getString(n int) (string, error) {
	if l.index+n > len(l.ucodes) {
		return string(l.ucodes[l.index:len(l.ucodes)]), errors.New("not enough char")
	}
	s := string(l.ucodes[l.index : l.index+n])
	return s, nil
}

func (l *Lexer) peekCharIs(ch rune) bool {
	nextIndex := l.index + 1
	if nextIndex >= len(l.ucodes) {
		return 0 == ch
	} else {
		return l.ucodes[nextIndex] == ch
	}
}

// 标记一个位置
func (l *Lexer) mark() {
	l.markIndex = l.index
	l.markPosition.Line = l.position.Line
	l.markPosition.Column = l.position.Column
}

func (l *Lexer) buildToken(ttype token.TokenType) token.Token {
	start := l.markPosition
	end := l.position
	startIndex := l.markIndex
	endIndex := l.index
	switch ttype {
	case token.STRING:
		// 移除首尾的引号
		startIndex++
		endIndex--
	case token.EOF:
		// 确保 EOF token 的 Literal 为空字符串
		endIndex = startIndex
	}
	tok := token.Token{Type: ttype, Literal: string(l.ucodes[startIndex:endIndex]), Start: start, End: end}
	return tok
}

func (l *Lexer) readIdentifier() token.Token {
	for isIdentifierContinue(l.ch) {
		l.readChar()
	}
	tok := l.buildToken(token.IDENT)
	tok.Type = token.LookupIdent(tok.Literal)
	return tok
}

func (l *Lexer) readNumber() token.Token {
	var buf []rune
	check := isDigit
	prefix, _ := l.getString(2)
	category := "decimal"
	switch prefix {
	case "0b", "0B":
		check = isBindigit
		buf = append(buf, '0', 'b')
		category = "binary"
		l.advance(2)
	case "0o", "0O":
		check = isOctDigit
		buf = append(buf, '0', 'o')
		category = "octal"
		l.advance(2)
	case "0x", "0X":
		check = isHexdigit
		buf = append(buf, '0', 'x')
		category = "hexadecimal"
		l.advance(2)
	}
	for {
		if l.ch == '_' {
			l.readChar()
			continue
		}
		if !isLetterAndDigit(l.ch) {
			break
		}
		if !check(l.ch) {
			tok := l.buildToken(token.ILLEGAL)
			tok.Literal = fmt.Sprintf("invalid digit '%s' in %s literal", string(l.ch), category)
			return tok
		}
		buf = append(buf, l.ch)
		l.readChar()
	}
	tok := l.buildToken(token.INT)
	tok.Literal = string(buf)
	if tok.Literal == "0b" || tok.Literal == "0o" || tok.Literal == "0x" {
		tok.Type = token.ILLEGAL
		tok.Literal = fmt.Sprintf("%s literal has no digits", category)
	}
	return tok
}

var escapeMap = map[rune]rune{
	'\\': '\\',
	'\'': '\'',
	'"':  '"',
	'a':  '\a',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
	'v':  '\v',
}

func parseRune(s string, base int, bitSize int) (rune, error) {
	n, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		return 0, err
	}
	r := rune(n)
	return r, nil
}

func (l *Lexer) readString(end rune) token.Token {
	var buf []rune
	// 跳过开始的引号
	l.readChar()
	for {
		// 参考 Python 的转义字符 https://docs.python.org/3/reference/lexical_analysis.html#string-and-bytes-literals
		// 处理转义字符
		ch := l.ch
		if ch == '\\' {
			l.readChar()
			if actual, ok := escapeMap[l.ch]; ok {
				buf = append(buf, actual)
				l.readChar()
				continue
			}
			// 解析 Unicode 转义字符
			var ucode rune
			var codeLen int
			switch l.ch {
			case 'x':
				// 格式为 "\xhh" h 代表十六进制字符
				// 跳过 'x' 字符
				l.readChar()
				codeLen = 2
				s, err := l.getString(codeLen)
				if err != nil {
					tok := l.buildToken(token.ILLEGAL)
					tok.Literal = "illegal escape sequence"
					return tok
				}
				ucode, err = parseRune(s, 16, 8)
				if err != nil {
					tok := l.buildToken(token.ILLEGAL)
					tok.Literal = "illegal escape sequence"
					return tok
				}
			case 'u':
				// 格式为 "\uhhhh" h 代表十六进制字符
				// 跳过 'u' 字符
				l.readChar()
				codeLen = 4
				s, err := l.getString(codeLen)
				if err != nil {
					tok := l.buildToken(token.ILLEGAL)
					tok.Literal = "illegal escape sequence"
					return tok
				}
				ucode, err = parseRune(s, 16, 16)
				if err != nil {
					tok := l.buildToken(token.ILLEGAL)
					tok.Literal = "illegal escape sequence"
					return tok
				}
			case 'U':
				// 格式为 "\Uhhhhhhhh" h 代表十六进制字符
				// 跳过 'u' 字符
				l.readChar()
				codeLen = 8
				s, err := l.getString(codeLen)
				if err != nil {
					tok := l.buildToken(token.ILLEGAL)
					tok.Literal = "illegal escape sequence"
					return tok
				}
				ucode, err = parseRune(s, 16, 32)
				if err != nil {
					tok := l.buildToken(token.ILLEGAL)
					tok.Literal = "illegal escape sequence"
					return tok
				}
			case '0', '1', '2', '3', '4', '5', '6', '7':
				// 格式为 "\ooo" o 代表八进制字符，最大为 "\377" (255)
				codeLen = 3
				s, err := l.getString(codeLen)
				if err != nil {
					tok := l.buildToken(token.ILLEGAL)
					tok.Literal = "illegal escape sequence"
					return tok
				}
				ucode, err = parseRune(s, 8, 8)
				if err != nil {
					tok := l.buildToken(token.ILLEGAL)
					tok.Literal = "illegal escape sequence"
					return tok
				}
			default:
				// 非法转义字符
				tok := l.buildToken(token.ILLEGAL)
				tok.Literal = "illegal escape sequence"
				return tok
			}
			if !utf8.ValidRune(ucode) {
				tok := l.buildToken(token.ILLEGAL)
				tok.Literal = "escape sequence is invalid Unicode code point"
				return tok
			}
			buf = append(buf, ucode)
			l.advance(codeLen)
			continue
		}

		if l.ch == end {
			break
		}
		if l.ch == 0 || l.ch == '\n' {
			tok := l.buildToken(token.ILLEGAL)
			tok.Literal = "string literal not terminated"
			return tok
		}
		buf = append(buf, l.ch)
		l.readChar()
	}
	// 跳过末尾的引号
	l.readChar()
	tok := l.buildToken(token.STRING)
	tok.Literal = string(buf)
	return tok
}

func (l *Lexer) readRawString() token.Token {
	// 跳过开始的引号
	l.readChar()
	for {
		if l.ch == 0 {
			tok := l.buildToken(token.ILLEGAL)
			tok.Literal = "string literal not terminated"
			return tok
		}
		if l.ch == '`' {
			break
		}
		l.readChar()
	}
	// 跳过结尾的引号
	l.readChar()
	return l.buildToken(token.STRING)
}

func (l *Lexer) readComment() token.Token {
	// 跳过开头的 // 两个字符
	l.readChar()
	l.readChar()
	l.mark()
	for {
		if l.ch == '\n' || l.ch == 0 {
			break
		}
		l.readChar()
	}
	return l.buildToken(token.COMMENT)
}

// 参考 Python 的规则 https://docs.python.org/3/reference/lexical_analysis.html#identifiers
var idStartCategorys = map[string]uint8{
	"Lu": 1,
	"Ll": 1,
	"Lm": 1,
	"Lt": 1,
	"Lo": 1,
	"Nl": 1,
}
var idContinueCategorys = map[string]uint8{
	"Lu": 1,
	"Ll": 1,
	"Lm": 1,
	"Lt": 1,
	"Lo": 1,
	"Nl": 1,
	"Mn": 1,
	"Mc": 1,
	"Nd": 1,
	"Pc": 1,
}

func isIdentifierStart(ch rune) bool {

	switch ch {
	case '_':
		return true
	default:
		cat := UnicodeCategory(ch)
		_, ok := idStartCategorys[cat]
		return ok
	}
}

func isIdentifierContinue(ch rune) bool {
	switch ch {
	case '_':
		return true
	default:
		cat := UnicodeCategory(ch)
		_, ok := idContinueCategorys[cat]
		return ok
	}
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func isBindigit(ch rune) bool {
	return ch == '0' || ch == '1'
}

func isOctDigit(ch rune) bool {
	return '0' <= ch && ch <= '7'
}

func isHexdigit(ch rune) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func isLetterAndDigit(ch rune) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

// UnicodeCategory returns the Unicode Character Category of the given rune.
// code from https://stackoverflow.com/a/53507592
func UnicodeCategory(r rune) string {
	for name, table := range unicode.Categories {
		if len(name) == 2 && unicode.Is(table, r) {
			return name
		}
	}
	return "Cn"
}
