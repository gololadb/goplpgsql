package parser

import (
	"strings"

	"github.com/gololadb/goplpgsql/scanner"
)

// readExprUntilSemi reads tokens until ';' at depth 0, returns the text.
// String literals are re-quoted so the expression is valid SQL.
func (p *Parser) readExprUntilSemi() string {
	return p.collectTokens(scanner.Token(';'), 0, 0, true)
}

// readExprUntilThen reads tokens until K_THEN at depth 0.
func (p *Parser) readExprUntilThen() string {
	return p.collectTokens(scanner.K_THEN, 0, 0, true)
}

// readExprUntilLoop reads tokens until K_LOOP at depth 0.
func (p *Parser) readExprUntilLoop() string {
	return p.collectTokens(scanner.K_LOOP, 0, 0, true)
}

// readSQLUntilSemi reads tokens until ';' at depth 0, returns the text.
// Does NOT consume the semicolon. String literals are re-quoted so the
// resulting text is valid SQL.
func (p *Parser) readSQLUntilSemi() string {
	return p.collectTokens(scanner.Token(';'), 0, 0, true)
}

// readSQLUntil reads tokens until one of up to 3 terminators at paren depth 0.
// The terminator is NOT consumed. Returns the collected text.
func (p *Parser) readSQLUntil(until1, until2, until3 scanner.Token) string {
	return p.collectTokens(until1, until2, until3, false)
}

// collectTokens reads tokens until one of up to 3 terminators at paren depth 0.
// When requote is true, string constants are re-wrapped in single quotes so the
// output is valid SQL.
func (p *Parser) collectTokens(until1, until2, until3 scanner.Token, requote bool) string {
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
		if requote && p.tok == scanner.SCONST {
			parts = append(parts, "'"+strings.ReplaceAll(p.lit, "'", "''")+"'")
		} else {
			parts = append(parts, p.lit)
		}
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
