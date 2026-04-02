package parser

import (
	"strings"

	"github.com/gololadb/goplpgsql/scanner"
)

// parseBlock parses: [<<label>>] [DECLARE decls] BEGIN stmts [EXCEPTION ...] END [label]
func (p *Parser) parseBlock() *StmtBlock {
	block := &StmtBlock{baseStmt: baseStmt{baseNode{Location: p.pos}}}

	// Optional label: <<label>>
	if p.tok == scanner.LESS_LESS {
		p.next()
		block.Label = p.anyIdentifier()
		p.want(scanner.GREATER_GREATER)
	}

	// Optional DECLARE section
	if p.gotKw(scanner.K_DECLARE) {
		block.Decls = p.parseDeclSection()
	}

	// BEGIN
	p.wantKw(scanner.K_BEGIN)

	// Statement list
	block.Body = p.parseProcSect()

	// Optional EXCEPTION section
	if p.isKw(scanner.K_EXCEPTION) {
		block.Exceptions = p.parseExceptionSect()
	}

	// END [label]
	p.wantKw(scanner.K_END)
	if p.tok == scanner.T_WORD || p.tok == scanner.IDENT || scanner.IsUnreservedKeyword(p.tok) {
		// optional end label
		p.next()
	}

	return block
}

// parseDeclSection parses declarations until BEGIN is seen.
func (p *Parser) parseDeclSection() []*DeclVar {
	var decls []*DeclVar
	for !p.isKw(scanner.K_BEGIN) && p.tok != scanner.EOF {
		// Allow extra DECLARE keywords
		if p.gotKw(scanner.K_DECLARE) {
			continue
		}
		// Block label inside DECLARE is an error in PG, but we skip it
		if p.tok == scanner.LESS_LESS {
			p.syntaxError("block label must be placed before DECLARE, not after")
			// skip past >>
			p.next()
			p.anyIdentifier()
			if p.tok == scanner.GREATER_GREATER {
				p.next()
			}
			continue
		}
		d := p.parseDeclStatement()
		if d != nil {
			decls = append(decls, d)
		}
	}
	return decls
}

// parseDeclStatement parses a single declaration.
func (p *Parser) parseDeclStatement() *DeclVar {
	d := &DeclVar{baseNode: baseNode{Location: p.pos}}

	// Variable name
	d.Name = p.anyIdentifier()

	// Check for ALIAS FOR
	if p.isKw(scanner.K_ALIAS) {
		p.next()
		p.wantKw(scanner.K_FOR)
		d.IsAlias = true
		d.AliasFor = p.anyIdentifier()
		p.wantSelf(';')
		return d
	}

	// Check for cursor declaration: name [NO] SCROLL CURSOR ...
	scrollOpt := ScrollDefault
	if p.isKw(scanner.K_NO) {
		// peek for SCROLL
		p.next()
		if p.isKw(scanner.K_SCROLL) {
			scrollOpt = ScrollNo
			p.next()
		} else {
			// Not NO SCROLL, push back and treat as type
			p.pushBack()
			// fall through to normal variable
			return p.parseDeclVarRest(d)
		}
	} else if p.isKw(scanner.K_SCROLL) {
		scrollOpt = ScrollYes
		p.next()
	}

	if p.isKw(scanner.K_CURSOR) {
		p.next()
		d.IsCursor = true
		d.ScrollOpt = scrollOpt

		// Optional cursor args: ( name type [, ...] )
		if p.gotSelf('(') {
			d.CursorArgs = p.parseCursorArgs()
			p.wantSelf(')')
		}

		// IS or FOR
		if !p.gotKw(scanner.K_IS) {
			p.wantKw(scanner.K_FOR)
		}

		// Cursor query: read until ;
		d.CursorQuery = p.readSQLUntilSemi()
		p.wantSelf(';')
		return d
	}

	// If we consumed a scroll option but no CURSOR follows, it was part of the type
	// This shouldn't normally happen, but handle gracefully
	if scrollOpt != ScrollDefault {
		// Put the keyword back conceptually - we already consumed it
		// Just treat as normal var declaration with type starting with "no"/"scroll"
		return p.parseDeclVarRest(d)
	}

	return p.parseDeclVarRest(d)
}

// parseDeclVarRest parses the rest of a variable declaration after the name.
func (p *Parser) parseDeclVarRest(d *DeclVar) *DeclVar {
	// Optional CONSTANT
	if p.gotKw(scanner.K_CONSTANT) {
		d.Constant = true
	}

	// Data type: read tokens until we hit COLLATE, NOT NULL, :=, =, DEFAULT, or ;
	d.DataType = p.readDataType()

	// Optional COLLATE
	if p.gotKw(scanner.K_COLLATE) {
		// consume collation name (we store it in the type for simplicity)
		collation := p.anyIdentifier()
		d.DataType += " COLLATE " + collation
	}

	// Optional NOT NULL
	if p.isKw(scanner.K_NOT) {
		p.next()
		p.wantKw(scanner.K_NULL)
		d.NotNull = true
	}

	// Optional default value
	if p.gotSelf(';') {
		return d
	}
	if p.gotSelf('=') || p.got(scanner.COLON_EQUALS) || p.gotKw(scanner.K_DEFAULT) {
		d.Default = p.readExprUntilSemi()
	}
	p.wantSelf(';')
	return d
}

// parseCursorArgs parses cursor argument declarations.
func (p *Parser) parseCursorArgs() []*CursorArg {
	var args []*CursorArg
	for {
		arg := &CursorArg{}
		arg.Name = p.anyIdentifier()
		arg.DataType = p.readDataTypeUntilCommaOrParen()
		args = append(args, arg)
		if !p.gotSelf(',') {
			break
		}
	}
	return args
}

// readDataType reads a data type specification. Types can be complex
// (e.g., "schema.table%ROWTYPE", "integer[]", "varchar(100)").
func (p *Parser) readDataType() string {
	var parts []string
	depth := 0
	for p.tok != scanner.EOF {
		// Stop at these tokens when not inside parens
		if depth == 0 {
			if p.tok == scanner.Token(';') {
				break
			}
			if p.tok == scanner.COLON_EQUALS || p.tok == scanner.Token('=') {
				break
			}
			if p.isKw(scanner.K_COLLATE) || p.isKw(scanner.K_NOT) || p.isKw(scanner.K_DEFAULT) {
				break
			}
		}
		if p.tok == scanner.Token('(') {
			depth++
		}
		if p.tok == scanner.Token(')') {
			if depth == 0 {
				break
			}
			depth--
		}
		parts = append(parts, p.lit)
		p.next()
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// readDataTypeUntilCommaOrParen reads a type until , or ) at depth 0.
func (p *Parser) readDataTypeUntilCommaOrParen() string {
	var parts []string
	depth := 0
	for p.tok != scanner.EOF {
		if depth == 0 {
			if p.tok == scanner.Token(',') || p.tok == scanner.Token(')') {
				break
			}
		}
		if p.tok == scanner.Token('(') {
			depth++
		}
		if p.tok == scanner.Token(')') {
			depth--
		}
		parts = append(parts, p.lit)
		p.next()
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
