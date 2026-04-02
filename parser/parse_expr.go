package parser

import (
	"strings"

	"github.com/gololadb/goplpgsql/scanner"
)

// readExprUntilSemi reads tokens until ';' at depth 0, returns the text.
func (p *Parser) readExprUntilSemi() string {
	return p.readSQLUntil(scanner.Token(';'), 0, 0)
}

// readExprUntilThen reads tokens until K_THEN at depth 0.
func (p *Parser) readExprUntilThen() string {
	return p.readSQLUntil(scanner.K_THEN, 0, 0)
}

// readExprUntilLoop reads tokens until K_LOOP at depth 0.
func (p *Parser) readExprUntilLoop() string {
	return p.readSQLUntil(scanner.K_LOOP, 0, 0)
}

// readSQLUntilSemi reads tokens until ';' at depth 0, returns the text.
// Does NOT consume the semicolon.
func (p *Parser) readSQLUntilSemi() string {
	return p.readSQLUntil(scanner.Token(';'), 0, 0)
}

// readSQLUntil reads tokens until one of up to 3 terminators at paren depth 0.
// The terminator is NOT consumed. Returns the collected text.
func (p *Parser) readSQLUntil(until1, until2, until3 scanner.Token) string {
	var parts []string
	depth := 0
	for p.tok != scanner.EOF {
		if depth == 0 {
			if p.tok == until1 {
				break
			}
			if until2 != 0 && p.tok == until2 {
				break
			}
			if until3 != 0 && p.tok == until3 {
				break
			}
		}
		if p.tok == scanner.Token('(') {
			depth++
		}
		if p.tok == scanner.Token(')') {
			if depth > 0 {
				depth--
			}
		}
		parts = append(parts, p.lit)
		p.next()
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// readSQLUntilEndToken reads tokens until one of the terminators, consuming it.
// Returns the text and which terminator was found.
func (p *Parser) readSQLUntilEndToken(until1, until2, until3 scanner.Token) (string, scanner.Token) {
	text := p.readSQLUntil(until1, until2, until3)
	endTok := p.tok
	if p.tok != scanner.EOF {
		p.next() // consume the terminator
	}
	return text, endTok
}
