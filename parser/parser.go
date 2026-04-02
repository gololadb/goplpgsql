package parser

import (
	"fmt"

	"github.com/gololadb/goplpgsql/scanner"
)

// Parser is a recursive-descent parser for PL/pgSQL.
type Parser struct {
	scan   scanner.Scanner
	errh   func(pos int, msg string)
	errcnt int
	first  error

	tok  scanner.Token
	lit  string
	cat  scanner.KeywordCategory
	pos  int
}

// Parse parses a PL/pgSQL function body and returns the top-level block.
func Parse(src []byte, errh func(pos int, msg string)) (*StmtBlock, error) {
	var p Parser
	p.init(src, errh)
	block := p.parseBlock()
	// consume optional trailing semicolon
	p.gotSelf(';')
	if p.tok != scanner.EOF {
		p.syntaxError("expected end of input")
	}
	return block, p.first
}

func (p *Parser) init(src []byte, errh func(pos int, msg string)) {
	p.errh = errh
	p.scan.Init(src, func(line, col uint, msg string) {
		if errh != nil {
			errh(-1, msg)
		}
	})
	p.next()
}

func (p *Parser) next() {
	p.scan.Next()
	p.tok = p.scan.Tok
	p.lit = p.scan.Lit
	p.cat = p.scan.Cat
	p.pos = int(p.scan.Line)*10000 + int(p.scan.Col)
}

func (p *Parser) pushBack() {
	p.scan.PushBack()
}

func (p *Parser) error(msg string) {
	err := fmt.Errorf("at position %d: %s", p.pos, msg)
	if p.first == nil {
		p.first = err
	}
	p.errcnt++
	if p.errh != nil {
		p.errh(p.pos, msg)
	}
}

func (p *Parser) errorf(format string, args ...any) {
	p.error(fmt.Sprintf(format, args...))
}

func (p *Parser) syntaxError(msg string) {
	if msg == "" {
		p.errorf("syntax error at %s", p.tokDesc())
	} else {
		p.errorf("syntax error: %s (got %s)", msg, p.tokDesc())
	}
}

func (p *Parser) tokDesc() string {
	switch {
	case p.tok == scanner.EOF:
		return "end of input"
	case p.tok == scanner.T_WORD:
		return fmt.Sprintf("identifier %q", p.lit)
	case p.tok == scanner.SCONST:
		return fmt.Sprintf("string '%s'", p.lit)
	case p.tok == scanner.ICONST:
		return fmt.Sprintf("integer %q", p.lit)
	case p.tok >= scanner.K_ALL && p.tok <= scanner.K_WARNING:
		return fmt.Sprintf("keyword %q", p.lit)
	case p.tok < 128:
		return fmt.Sprintf("'%c'", rune(p.tok))
	default:
		return fmt.Sprintf("token(%d) %q", p.tok, p.lit)
	}
}

// got consumes the current token if it matches and returns true.
func (p *Parser) got(tok scanner.Token) bool {
	if p.tok == tok {
		p.next()
		return true
	}
	return false
}

// want consumes the current token if it matches, or reports an error.
func (p *Parser) want(tok scanner.Token) {
	if !p.got(tok) {
		p.syntaxError(fmt.Sprintf("expected %s", tokName(tok)))
	}
}

// gotSelf consumes a single-character token.
func (p *Parser) gotSelf(ch rune) bool {
	if p.tok == scanner.Token(ch) {
		p.next()
		return true
	}
	return false
}

// wantSelf consumes a single-character token or reports an error.
func (p *Parser) wantSelf(ch rune) {
	if !p.gotSelf(ch) {
		p.syntaxError(fmt.Sprintf("expected '%c'", ch))
	}
}

// gotKw consumes a keyword token and returns true.
func (p *Parser) gotKw(tok scanner.Token) bool {
	if p.tok == tok {
		p.next()
		return true
	}
	return false
}

// wantKw consumes a keyword or reports an error.
func (p *Parser) wantKw(tok scanner.Token) {
	if !p.gotKw(tok) {
		p.syntaxError(fmt.Sprintf("expected keyword"))
	}
}

// isKw returns true if the current token is the given keyword.
func (p *Parser) isKw(tok scanner.Token) bool {
	return p.tok == tok
}

// isAnyKw returns true if the current token is any of the given keywords.
func (p *Parser) isAnyKw(toks ...scanner.Token) bool {
	for _, t := range toks {
		if p.tok == t {
			return true
		}
	}
	return false
}

// anyIdentifier consumes an identifier, unreserved keyword, or PARAM and returns its text.
func (p *Parser) anyIdentifier() string {
	if p.tok == scanner.T_WORD || p.tok == scanner.T_CWORD || p.tok == scanner.IDENT || p.tok == scanner.PARAM {
		name := p.lit
		p.next()
		return name
	}
	if scanner.IsUnreservedKeyword(p.tok) {
		name := p.lit
		p.next()
		return name
	}
	p.syntaxError("expected identifier")
	p.next()
	return ""
}

func tokName(tok scanner.Token) string {
	if tok < 128 {
		return fmt.Sprintf("'%c'", rune(tok))
	}
	switch tok {
	case scanner.EOF:
		return "end of input"
	case scanner.T_WORD:
		return "identifier"
	case scanner.ICONST:
		return "integer"
	case scanner.SCONST:
		return "string"
	default:
		return fmt.Sprintf("token(%d)", tok)
	}
}
