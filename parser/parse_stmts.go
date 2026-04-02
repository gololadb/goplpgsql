package parser

import "github.com/gololadb/goplpgsql/scanner"

// parseProcSect parses a list of statements until END, EXCEPTION, ELSE, ELSIF, or WHEN.
func (p *Parser) parseProcSect() []Stmt {
	var stmts []Stmt
	for !p.isAnyKw(scanner.K_END, scanner.K_EXCEPTION, scanner.K_ELSE, scanner.K_ELSIF, scanner.K_WHEN) && p.tok != scanner.EOF {
		s := p.parseProcStmt()
		if s != nil {
			stmts = append(stmts, s)
		}
	}
	return stmts
}

// parseProcStmt parses a single PL/pgSQL statement.
func (p *Parser) parseProcStmt() Stmt {
	// Label: <<label>> followed by LOOP, WHILE, FOR, FOREACH, DECLARE, or BEGIN
	if p.tok == scanner.LESS_LESS {
		p.next()
		label := p.anyIdentifier()
		p.want(scanner.GREATER_GREATER)

		switch p.tok {
		case scanner.K_LOOP:
			return p.parseStmtLoop(label)
		case scanner.K_WHILE:
			return p.parseStmtWhile(label)
		case scanner.K_FOR:
			return p.parseStmtFor(label)
		case scanner.K_FOREACH:
			return p.parseStmtForEach(label)
		case scanner.K_DECLARE, scanner.K_BEGIN:
			// Labeled nested block - push label back into block parsing
			block := &StmtBlock{baseStmt: baseStmt{baseNode{Location: p.pos}}, Label: label}
			if p.gotKw(scanner.K_DECLARE) {
				block.Decls = p.parseDeclSection()
			}
			p.wantKw(scanner.K_BEGIN)
			block.Body = p.parseProcSect()
			if p.isKw(scanner.K_EXCEPTION) {
				block.Exceptions = p.parseExceptionSect()
			}
			p.wantKw(scanner.K_END)
			if p.tok == scanner.T_WORD || p.tok == scanner.IDENT || scanner.IsUnreservedKeyword(p.tok) {
				p.next()
			}
			p.wantSelf(';')
			return block
		default:
			p.syntaxError("expected LOOP, WHILE, FOR, FOREACH, DECLARE, or BEGIN after label")
			return nil
		}
	}

	// Nested block without label
	if p.isKw(scanner.K_DECLARE) || p.isKw(scanner.K_BEGIN) {
		block := p.parseBlock()
		p.wantSelf(';')
		return block
	}

	switch p.tok {
	case scanner.K_IF:
		return p.parseStmtIf()
	case scanner.K_CASE:
		return p.parseStmtCase()
	case scanner.K_LOOP:
		return p.parseStmtLoop("")
	case scanner.K_WHILE:
		return p.parseStmtWhile("")
	case scanner.K_FOR:
		return p.parseStmtFor("")
	case scanner.K_FOREACH:
		return p.parseStmtForEach("")
	case scanner.K_EXIT:
		return p.parseStmtExit(true)
	case scanner.K_CONTINUE:
		return p.parseStmtExit(false)
	case scanner.K_RETURN:
		return p.parseStmtReturn()
	case scanner.K_RAISE:
		return p.parseStmtRaise()
	case scanner.K_ASSERT:
		return p.parseStmtAssert()
	case scanner.K_EXECUTE:
		return p.parseStmtDynExecute()
	case scanner.K_PERFORM:
		return p.parseStmtPerform()
	case scanner.K_CALL:
		return p.parseStmtCall(true)
	case scanner.K_DO:
		return p.parseStmtCall(false)
	case scanner.K_GET:
		return p.parseStmtGetDiag()
	case scanner.K_OPEN:
		return p.parseStmtOpen()
	case scanner.K_FETCH:
		return p.parseStmtFetch(false)
	case scanner.K_MOVE:
		return p.parseStmtFetch(true)
	case scanner.K_CLOSE:
		return p.parseStmtClose()
	case scanner.K_NULL:
		return p.parseStmtNull()
	case scanner.K_COMMIT:
		return p.parseStmtCommit()
	case scanner.K_ROLLBACK:
		return p.parseStmtRollback()
	case scanner.K_INSERT, scanner.K_IMPORT, scanner.K_MERGE:
		return p.parseStmtExecSQL()
	}

	// T_WORD or T_CWORD: could be assignment or SQL statement
	if p.tok == scanner.T_WORD || p.tok == scanner.T_CWORD || p.tok == scanner.IDENT {
		return p.parseStmtWordStart()
	}

	// Unreserved keywords that start SQL statements
	if scanner.IsUnreservedKeyword(p.tok) {
		return p.parseStmtExecSQL()
	}

	p.syntaxError("expected statement")
	p.next()
	return nil
}

// parseStmtWordStart handles statements starting with an identifier.
// Could be assignment (var := expr) or SQL (SELECT, UPDATE, etc).
func (p *Parser) parseStmtWordStart() Stmt {
	pos := p.pos
	name := p.lit
	p.next()

	// Check for label: identifier followed by <<
	// Actually labels use <<label>> before LOOP/WHILE/FOR
	// Check for assignment operators
	if p.tok == scanner.COLON_EQUALS || p.tok == scanner.Token('=') {
		p.next() // consume := or =
		expr := p.readExprUntilSemi()
		p.wantSelf(';')
		return &StmtAssign{
			baseStmt: baseStmt{baseNode{Location: pos}},
			Variable: name,
			Expr:     expr,
		}
	}

	// Check for dotted name assignment: a.b.c := expr
	if p.tok == scanner.Token('.') {
		fullName := name
		for p.gotSelf('.') {
			fullName += "." + p.lit
			p.next()
		}
		if p.tok == scanner.COLON_EQUALS || p.tok == scanner.Token('=') {
			p.next()
			expr := p.readExprUntilSemi()
			p.wantSelf(';')
			return &StmtAssign{
				baseStmt: baseStmt{baseNode{Location: pos}},
				Variable: fullName,
				Expr:     expr,
			}
		}
		// Not assignment, treat as SQL
		return p.finishExecSQL(pos, fullName)
	}

	// Check for array subscript assignment: a[i] := expr
	if p.tok == scanner.Token('[') {
		fullName := name
		// Read through subscripts
		for p.tok == scanner.Token('[') {
			fullName += " ["
			p.next()
			depth := 1
			for depth > 0 && p.tok != scanner.EOF {
				if p.tok == scanner.Token('[') {
					depth++
				}
				if p.tok == scanner.Token(']') {
					depth--
					if depth == 0 {
						break
					}
				}
				fullName += " " + p.lit
				p.next()
			}
			fullName += " ]"
			if p.tok == scanner.Token(']') {
				p.next()
			}
		}
		if p.tok == scanner.COLON_EQUALS || p.tok == scanner.Token('=') {
			p.next()
			expr := p.readExprUntilSemi()
			p.wantSelf(';')
			return &StmtAssign{
				baseStmt: baseStmt{baseNode{Location: pos}},
				Variable: fullName,
				Expr:     expr,
			}
		}
		return p.finishExecSQL(pos, fullName)
	}

	// Otherwise it's a SQL statement
	return p.finishExecSQL(pos, name)
}

// finishExecSQL reads the rest of a SQL statement given the first word(s).
func (p *Parser) finishExecSQL(pos int, prefix string) Stmt {
	rest := p.readSQLUntilSemi()
	sql := prefix
	if rest != "" {
		sql += " " + rest
	}
	p.wantSelf(';')

	stmt := &StmtExecSQL{
		baseStmt: baseStmt{baseNode{Location: pos}},
		SQL:      sql,
	}
	return stmt
}

// parseStmtExecSQL parses a SQL statement (INSERT, UPDATE, SELECT, etc).
func (p *Parser) parseStmtExecSQL() Stmt {
	pos := p.pos
	firstWord := p.lit
	p.next()
	return p.finishExecSQL(pos, firstWord)
}
