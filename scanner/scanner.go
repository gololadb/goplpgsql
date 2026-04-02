package scanner

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Scanner is a lexical tokenizer for PL/pgSQL source.
// After initialization, consecutive calls to Next advance one token at a time.
type Scanner struct {
	src    []byte
	offset int // current read position
	ch     rune
	chw    int

	// Current token state, valid after calling Next().
	Line uint // 1-based line
	Col  uint // 1-based column
	Tok  Token
	Lit  string // token text
	Cat  KeywordCategory

	// Position tracking
	line uint // 0-based current line
	col  uint // 0-based current column

	errh func(line, col uint, msg string)

	// pushback support
	hasPushback bool
	pbTok       Token
	pbLit       string
	pbCat       KeywordCategory
	pbLine      uint
	pbCol       uint
}

// Init initializes the scanner with source bytes.
func (s *Scanner) Init(src []byte, errh func(line, col uint, msg string)) {
	s.src = src
	s.offset = 0
	s.line = 0
	s.col = 0
	s.errh = errh
	s.hasPushback = false
	// Prime the first character
	if len(src) > 0 {
		s.ch, s.chw = utf8.DecodeRune(src)
		if s.ch == utf8.RuneError && s.chw == 1 {
			s.ch = rune(src[0])
			s.chw = 1
		}
	} else {
		s.ch = -1
		s.chw = 0
	}
}

// PushBack pushes the current token back so the next call to Next returns it again.
func (s *Scanner) PushBack() {
	s.hasPushback = true
	s.pbTok = s.Tok
	s.pbLit = s.Lit
	s.pbCat = s.Cat
	s.pbLine = s.Line
	s.pbCol = s.Col
}

func (s *Scanner) error(msg string) {
	if s.errh != nil {
		s.errh(s.line+1, s.col+1, msg)
	}
}

func (s *Scanner) errorf(format string, args ...any) {
	s.error(fmt.Sprintf(format, args...))
}

func (s *Scanner) nextch() {
	if s.ch == '\n' {
		s.line++
		s.col = 0
	} else {
		s.col += uint(s.chw)
	}
	s.offset += s.chw
	if s.offset >= len(s.src) {
		s.ch = -1
		s.chw = 0
		return
	}
	s.ch, s.chw = utf8.DecodeRune(s.src[s.offset:])
	if s.ch == utf8.RuneError && s.chw == 1 {
		s.ch = rune(s.src[s.offset])
		s.chw = 1
	}
}

func isSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' || ch == '\v'
}

func isDigit(ch rune) bool  { return '0' <= ch && ch <= '9' }
func isHexDig(ch rune) bool { return isDigit(ch) || 'a' <= lower(ch) && lower(ch) <= 'f' }
func lower(ch rune) rune    { return ('a' - 'A') | ch }

func isIdentStart(ch rune) bool {
	if ch >= utf8.RuneSelf {
		return unicode.IsLetter(ch)
	}
	return 'a' <= lower(ch) && lower(ch) <= 'z' || ch == '_'
}

func isIdentCont(ch rune) bool {
	if ch >= utf8.RuneSelf {
		return unicode.IsLetter(ch) || unicode.IsDigit(ch)
	}
	return isIdentStart(ch) || isDigit(ch) || ch == '$'
}

// Next advances the scanner by one token.
func (s *Scanner) Next() {
	if s.hasPushback {
		s.hasPushback = false
		s.Tok = s.pbTok
		s.Lit = s.pbLit
		s.Cat = s.pbCat
		s.Line = s.pbLine
		s.Col = s.pbCol
		return
	}

redo:
	// Skip whitespace
	for isSpace(s.ch) {
		s.nextch()
	}

	// Skip -- line comments
	if s.ch == '-' {
		off := s.offset
		s.nextch()
		if s.ch == '-' {
			for s.ch >= 0 && s.ch != '\n' {
				s.nextch()
			}
			goto redo
		}
		// Not a comment, it's a minus sign or operator
		s.Line = s.line + 1
		s.Col = s.col
		// Check for ->
		if s.ch == '>' {
			s.nextch()
			s.Tok = Op
			s.Lit = "->"
			return
		}
		s.Tok = Token('-')
		s.Lit = string(s.src[off:s.offset])
		if s.Lit == "" {
			s.Lit = "-"
		}
		return
	}

	// Skip /* block comments */
	if s.ch == '/' {
		s.nextch()
		if s.ch == '*' {
			s.nextch()
			depth := 1
			for depth > 0 && s.ch >= 0 {
				if s.ch == '/' {
					s.nextch()
					if s.ch == '*' {
						depth++
						s.nextch()
					}
				} else if s.ch == '*' {
					s.nextch()
					if s.ch == '/' {
						depth--
						s.nextch()
					}
				} else {
					s.nextch()
				}
			}
			goto redo
		}
		s.Line = s.line + 1
		s.Col = s.col
		s.Tok = Token('/')
		s.Lit = "/"
		return
	}

	// Record token position
	s.Line = s.line + 1
	s.Col = s.col + 1

	// EOF
	if s.ch < 0 {
		s.Tok = EOF
		s.Lit = ""
		return
	}

	// Identifiers and keywords
	if isIdentStart(s.ch) {
		s.scanIdent()
		return
	}

	// Numbers
	if isDigit(s.ch) {
		s.scanNumber()
		return
	}

	// Dot
	if s.ch == '.' {
		s.nextch()
		if isDigit(s.ch) {
			s.scanNumberAfterDot()
			return
		}
		if s.ch == '.' {
			s.nextch()
			s.Tok = DOT_DOT
			s.Lit = ".."
			return
		}
		s.Tok = Token('.')
		s.Lit = "."
		return
	}

	// String literals
	if s.ch == '\'' {
		s.scanString()
		return
	}

	// Double-quoted identifiers
	if s.ch == '"' {
		s.scanQuotedIdent()
		return
	}

	// Dollar-quoted strings
	if s.ch == '$' {
		s.nextch()
		if isDigit(s.ch) {
			s.scanParam()
			return
		}
		if s.ch == '$' || isIdentStart(s.ch) {
			s.scanDollarString()
			return
		}
		s.Tok = Token('$')
		s.Lit = "$"
		return
	}

	// Typecast :: and := and bare :
	if s.ch == ':' {
		s.nextch()
		if s.ch == ':' {
			s.nextch()
			s.Tok = TYPECAST
			s.Lit = "::"
			return
		}
		if s.ch == '=' {
			s.nextch()
			s.Tok = COLON_EQUALS
			s.Lit = ":="
			return
		}
		s.Tok = Token(':')
		s.Lit = ":"
		return
	}

	// << and >>
	if s.ch == '<' {
		s.nextch()
		if s.ch == '<' {
			s.nextch()
			s.Tok = LESS_LESS
			s.Lit = "<<"
			return
		}
		if s.ch == '=' {
			s.nextch()
			s.Tok = LESS_EQUALS
			s.Lit = "<="
			return
		}
		if s.ch == '>' {
			s.nextch()
			s.Tok = NOT_EQUALS
			s.Lit = "<>"
			return
		}
		s.Tok = Token('<')
		s.Lit = "<"
		return
	}

	if s.ch == '>' {
		s.nextch()
		if s.ch == '>' {
			s.nextch()
			s.Tok = GREATER_GREATER
			s.Lit = ">>"
			return
		}
		if s.ch == '=' {
			s.nextch()
			s.Tok = GREATER_EQUALS
			s.Lit = ">="
			return
		}
		s.Tok = Token('>')
		s.Lit = ">"
		return
	}

	// = and =>
	if s.ch == '=' {
		s.nextch()
		if s.ch == '>' {
			s.nextch()
			s.Tok = EQUALS_GREATER
			s.Lit = "=>"
			return
		}
		s.Tok = Token('=')
		s.Lit = "="
		return
	}

	// != 
	if s.ch == '!' {
		s.nextch()
		if s.ch == '=' {
			s.nextch()
			s.Tok = NOT_EQUALS
			s.Lit = "!="
			return
		}
		s.Tok = Op
		s.Lit = "!"
		return
	}

	// # (used for compiler options)
	if s.ch == '#' {
		s.nextch()
		s.Tok = Token('#')
		s.Lit = "#"
		return
	}

	// Operators and other multi-char tokens
	if isOpChar(s.ch) {
		s.scanOperator()
		return
	}

	// Single-character self tokens
	ch := s.ch
	s.nextch()
	s.Tok = Token(ch)
	s.Lit = string(ch)
}

func isOpChar(ch rune) bool {
	return strings.ContainsRune("~!@#^&|`?+-*/%<>=", ch)
}

func (s *Scanner) scanIdent() {
	start := s.offset
	for isIdentCont(s.ch) {
		s.nextch()
	}
	word := string(s.src[start:s.offset])

	// Check for composite word (a.b.c) BEFORE keyword lookup,
	// because keywords followed by '.' are composite identifiers, not keywords.
	if s.ch == '.' {
		if parts, ok := s.tryComposite(word); ok {
			s.Tok = T_CWORD
			s.Lit = strings.Join(parts, ".")
			return
		}
	}

	// Check for PL/pgSQL keywords
	if tok, cat, ok := LookupKeyword(word); ok {
		s.Tok = tok
		s.Lit = strings.ToLower(word)
		s.Cat = cat
		return
	}

	s.Tok = T_WORD
	s.Lit = word
}

// tryComposite attempts to scan a dotted identifier (a.b.c).
// Returns the parts and true if successful, or restores state and returns false.
func (s *Scanner) tryComposite(firstWord string) ([]string, bool) {
	savedOffset := s.offset
	savedCh := s.ch
	savedChw := s.chw
	savedLine := s.line
	savedCol := s.col

	s.nextch() // consume '.'
	if !isIdentStart(s.ch) {
		// Not a composite, restore
		s.offset = savedOffset
		s.ch = savedCh
		s.chw = savedChw
		s.line = savedLine
		s.col = savedCol
		return nil, false
	}

	parts := []string{firstWord}
	for {
		partStart := s.offset
		for isIdentCont(s.ch) {
			s.nextch()
		}
		parts = append(parts, string(s.src[partStart:s.offset]))
		if s.ch != '.' {
			break
		}
		// Peek ahead
		peekOff := s.offset
		peekCh := s.ch
		peekChw := s.chw
		peekLine := s.line
		peekCol := s.col
		s.nextch()
		if !isIdentStart(s.ch) {
			s.offset = peekOff
			s.ch = peekCh
			s.chw = peekChw
			s.line = peekLine
			s.col = peekCol
			break
		}
	}
	return parts, true
}

func (s *Scanner) scanNumber() {
	start := s.offset
	for isDigit(s.ch) {
		s.nextch()
	}
	isFloat := false
	if s.ch == '.' {
		// Check for .. (DOT_DOT)
		next := s.offset + s.chw
		if next < len(s.src) && s.src[next] == '.' {
			// It's .., so the number ends here
			s.Tok = ICONST
			s.Lit = string(s.src[start:s.offset])
			return
		}
		isFloat = true
		s.nextch()
		for isDigit(s.ch) {
			s.nextch()
		}
	}
	if s.ch == 'e' || s.ch == 'E' {
		isFloat = true
		s.nextch()
		if s.ch == '+' || s.ch == '-' {
			s.nextch()
		}
		for isDigit(s.ch) {
			s.nextch()
		}
	}
	if isFloat {
		s.Tok = FCONST
	} else {
		s.Tok = ICONST
	}
	s.Lit = string(s.src[start:s.offset])
}

func (s *Scanner) scanNumberAfterDot() {
	start := s.offset - 1 // include the dot
	for isDigit(s.ch) {
		s.nextch()
	}
	if s.ch == 'e' || s.ch == 'E' {
		s.nextch()
		if s.ch == '+' || s.ch == '-' {
			s.nextch()
		}
		for isDigit(s.ch) {
			s.nextch()
		}
	}
	s.Tok = FCONST
	s.Lit = string(s.src[start:s.offset])
}

func (s *Scanner) scanString() {
	// s.ch is '\''
	var buf []byte
	s.nextch() // consume opening quote
	for {
		if s.ch < 0 {
			s.error("unterminated string literal")
			break
		}
		if s.ch == '\'' {
			s.nextch()
			if s.ch == '\'' {
				// escaped quote
				buf = append(buf, '\'')
				s.nextch()
				continue
			}
			break
		}
		buf = append(buf, s.src[s.offset:s.offset+s.chw]...)
		s.nextch()
	}
	s.Tok = SCONST
	s.Lit = string(buf)
}

func (s *Scanner) scanQuotedIdent() {
	// s.ch is '"'
	s.nextch() // consume opening quote
	start := s.offset
	for {
		if s.ch < 0 {
			s.error("unterminated quoted identifier")
			break
		}
		if s.ch == '"' {
			s.nextch()
			if s.ch == '"' {
				// escaped quote, continue
				s.nextch()
				continue
			}
			break
		}
		s.nextch()
	}
	s.Tok = IDENT
	s.Lit = string(s.src[start : s.offset-1]) // exclude closing quote
}

func (s *Scanner) scanParam() {
	// s.ch is first digit after $
	start := s.offset - 1 // include $
	for isDigit(s.ch) {
		s.nextch()
	}
	s.Tok = PARAM
	s.Lit = string(s.src[start:s.offset])
}

// isDollarTagCont returns true for characters valid in a dollar-quote tag.
// Unlike isIdentCont, this excludes '$' to avoid consuming the closing delimiter.
func isDollarTagCont(ch rune) bool {
	return isIdentStart(ch) || isDigit(ch)
}

func (s *Scanner) scanDollarString() {
	// We're after the initial $, s.ch is either $ or an ident start
	var tag string
	if s.ch == '$' {
		tag = ""
		s.nextch() // consume closing $ of empty tag
	} else {
		tagStart := s.offset
		for isDollarTagCont(s.ch) {
			s.nextch()
		}
		if s.ch != '$' {
			// Not a valid dollar-quote, return as bare $
			s.Tok = Token('$')
			s.Lit = "$"
			return
		}
		tag = string(s.src[tagStart:s.offset])
		s.nextch() // consume closing $ of tag
	}

	// Now scan body until we find the closing $tag$
	delim := "$" + tag + "$"
	bodyStart := s.offset
	for s.ch >= 0 {
		if s.ch == '$' {
			// Check if the closing delimiter starts here
			end := s.offset + len(delim)
			if end <= len(s.src) && string(s.src[s.offset:end]) == delim {
				body := string(s.src[bodyStart:s.offset])
				// Advance past the delimiter
				for i := 0; i < len(delim); i++ {
					s.nextch()
				}
				s.Tok = SCONST
				s.Lit = body
				return
			}
		}
		s.nextch()
	}
	s.error("unterminated dollar-quoted string")
	s.Tok = SCONST
	s.Lit = string(s.src[bodyStart:s.offset])
}

func (s *Scanner) scanOperator() {
	start := s.offset
	for isOpChar(s.ch) {
		s.nextch()
	}
	s.Tok = Op
	s.Lit = string(s.src[start:s.offset])
}
