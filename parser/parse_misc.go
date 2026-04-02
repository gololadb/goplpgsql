package parser

import (
	"strings"

	"github.com/gololadb/goplpgsql/scanner"
)

// parseStmtReturn parses: RETURN [NEXT|QUERY] [expr] ;
func (p *Parser) parseStmtReturn() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_RETURN)

	// RETURN NEXT expr ;
	if p.isKw(scanner.K_NEXT) {
		p.next()
		expr := p.readExprUntilSemi()
		p.wantSelf(';')
		return &StmtReturnNext{
			baseStmt: baseStmt{baseNode{Location: pos}},
			Expr:     expr,
		}
	}

	// RETURN QUERY [EXECUTE] query ;
	if p.isKw(scanner.K_QUERY) {
		p.next()
		if p.isKw(scanner.K_EXECUTE) {
			p.next()
			query, endTok := p.readSQLUntilEndToken(scanner.Token(';'), scanner.K_USING, 0)
			var params []string
			if endTok == scanner.K_USING {
				for {
					param, et := p.readSQLUntilEndToken(scanner.Token(','), scanner.Token(';'), 0)
					params = append(params, param)
					if et != scanner.Token(',') {
						break
					}
				}
			}
			return &StmtReturnQuery{
				baseStmt:  baseStmt{baseNode{Location: pos}},
				DynQuery:  query,
				DynParams: params,
			}
		}
		query := p.readExprUntilSemi()
		p.wantSelf(';')
		return &StmtReturnQuery{
			baseStmt: baseStmt{baseNode{Location: pos}},
			Query:    query,
		}
	}

	// RETURN [expr] ;
	if p.tok == scanner.Token(';') {
		p.next()
		return &StmtReturn{baseStmt: baseStmt{baseNode{Location: pos}}}
	}
	expr := p.readExprUntilSemi()
	p.wantSelf(';')
	return &StmtReturn{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Expr:     expr,
	}
}

// parseStmtRaise parses RAISE statements.
func (p *Parser) parseStmtRaise() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_RAISE)

	stmt := &StmtRaise{baseStmt: baseStmt{baseNode{Location: pos}}}

	// Bare RAISE ; (re-raise)
	if p.tok == scanner.Token(';') {
		p.next()
		return stmt
	}

	// Optional level
	switch p.tok {
	case scanner.K_EXCEPTION:
		stmt.Level = "EXCEPTION"
		p.next()
	case scanner.K_WARNING:
		stmt.Level = "WARNING"
		p.next()
	case scanner.K_NOTICE:
		stmt.Level = "NOTICE"
		p.next()
	case scanner.K_INFO:
		stmt.Level = "INFO"
		p.next()
	case scanner.K_LOG:
		stmt.Level = "LOG"
		p.next()
	case scanner.K_DEBUG:
		stmt.Level = "DEBUG"
		p.next()
	}

	if p.tok == scanner.Token(';') {
		p.next()
		return stmt
	}

	// String literal = format message
	if p.tok == scanner.SCONST {
		stmt.Message = p.lit
		p.next()

		// Comma-separated parameters: RAISE level 'fmt', expr1, expr2 [USING ...] ;
		if p.gotSelf(',') {
			for {
				if p.isKw(scanner.K_USING) || p.tok == scanner.Token(';') {
					break
				}
				param, endTok := p.readSQLUntilEndToken(scanner.Token(','), scanner.Token(';'), scanner.K_USING)
				stmt.Params = append(stmt.Params, param)
				if endTok == scanner.K_USING {
					stmt.Options = p.parseRaiseOptions()
					return stmt
				}
				if endTok != scanner.Token(',') {
					// endTok is ';'
					return stmt
				}
				// endTok is ',' — continue to next param
			}
		}

		if p.isKw(scanner.K_USING) {
			p.next()
			stmt.Options = p.parseRaiseOptions()
			return stmt
		}

		p.wantSelf(';')
		return stmt
	}

	// SQLSTATE 'xxxxx' or condition name
	if p.isKw(scanner.K_SQLSTATE) {
		p.next()
		if p.tok == scanner.SCONST {
			stmt.CondName = p.lit
			p.next()
		}
	} else if p.tok == scanner.T_WORD || scanner.IsUnreservedKeyword(p.tok) {
		// Check if it's USING (which means no condition)
		if !p.isKw(scanner.K_USING) {
			stmt.CondName = p.lit
			p.next()
		}
	}

	if p.isKw(scanner.K_USING) {
		p.next()
		stmt.Options = p.parseRaiseOptions()
		return stmt
	}

	p.wantSelf(';')
	return stmt
}

// parseRaiseOptions parses USING option = expr [, ...]
func (p *Parser) parseRaiseOptions() []*RaiseOption {
	var opts []*RaiseOption
	for {
		opt := &RaiseOption{}
		// option name
		opt.OptType = strings.ToUpper(p.lit)
		p.next()
		// = or :=
		if !p.gotSelf('=') {
			p.got(scanner.COLON_EQUALS)
		}
		expr, endTok := p.readSQLUntilEndToken(scanner.Token(','), scanner.Token(';'), 0)
		opt.Expr = expr
		opts = append(opts, opt)
		if endTok != scanner.Token(',') {
			break
		}
	}
	return opts
}

// parseStmtAssert parses: ASSERT condition [, message] ;
func (p *Parser) parseStmtAssert() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_ASSERT)

	cond, endTok := p.readSQLUntilEndToken(scanner.Token(','), scanner.Token(';'), 0)

	var msg string
	if endTok == scanner.Token(',') {
		msg = p.readExprUntilSemi()
		p.wantSelf(';')
	}

	return &StmtAssert{
		baseStmt:  baseStmt{baseNode{Location: pos}},
		Condition: cond,
		Message:   msg,
	}
}

// parseStmtDynExecute parses: EXECUTE expr [INTO [STRICT] target] [USING params] ;
func (p *Parser) parseStmtDynExecute() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_EXECUTE)

	query, endTok := p.readSQLUntilEndToken(scanner.K_INTO, scanner.K_USING, scanner.Token(';'))

	stmt := &StmtDynExecute{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Query:    query,
	}

	// Process INTO and USING in any order
	for endTok != scanner.Token(';') && endTok != scanner.EOF {
		if endTok == scanner.K_INTO {
			stmt.Into = true
			if p.isKw(scanner.K_STRICT) {
				stmt.Strict = true
				p.next()
			}
			target, et := p.readSQLUntilEndToken(scanner.K_USING, scanner.Token(';'), 0)
			stmt.Target = target
			endTok = et
		} else if endTok == scanner.K_USING {
			for {
				param, et := p.readSQLUntilEndToken(scanner.Token(','), scanner.Token(';'), scanner.K_INTO)
				stmt.Params = append(stmt.Params, param)
				if et != scanner.Token(',') {
					endTok = et
					break
				}
			}
		} else {
			break
		}
	}

	return stmt
}

// parseStmtPerform parses: PERFORM expr ;
func (p *Parser) parseStmtPerform() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_PERFORM)
	expr := p.readExprUntilSemi()
	p.wantSelf(';')
	return &StmtPerform{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Expr:     expr,
	}
}

// parseStmtCall parses: CALL proc(...) ; or DO $$ ... $$ ;
func (p *Parser) parseStmtCall(isCall bool) Stmt {
	pos := p.pos
	p.next() // consume CALL or DO
	expr := p.readExprUntilSemi()
	p.wantSelf(';')
	return &StmtCall{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Expr:     expr,
		IsCall:   isCall,
	}
}

// parseStmtGetDiag parses: GET [CURRENT|STACKED] DIAGNOSTICS target = item [, ...] ;
func (p *Parser) parseStmtGetDiag() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_GET)

	isStacked := false
	if p.isKw(scanner.K_STACKED) {
		isStacked = true
		p.next()
	} else if p.isKw(scanner.K_CURRENT) {
		p.next()
	}

	p.wantKw(scanner.K_DIAGNOSTICS)

	var items []*DiagItem
	for {
		item := &DiagItem{}
		item.Target = p.anyIdentifier()
		// = or :=
		if !p.gotSelf('=') {
			p.got(scanner.COLON_EQUALS)
		}
		item.Kind = strings.ToUpper(p.lit)
		p.next()
		items = append(items, item)
		if !p.gotSelf(',') {
			break
		}
	}

	p.wantSelf(';')

	return &StmtGetDiag{
		baseStmt:  baseStmt{baseNode{Location: pos}},
		IsStacked: isStacked,
		Items:     items,
	}
}

// parseStmtOpen parses OPEN cursor statements.
func (p *Parser) parseStmtOpen() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_OPEN)

	curVar := p.anyIdentifier()

	stmt := &StmtOpen{
		baseStmt: baseStmt{baseNode{Location: pos}},
		CurVar:   curVar,
	}

	// Check for scroll options and FOR (unbound cursor)
	if p.isKw(scanner.K_NO) {
		p.next()
		if p.isKw(scanner.K_SCROLL) {
			stmt.ScrollOpt = ScrollNo
			p.next()
		} else {
			p.pushBack()
		}
	} else if p.isKw(scanner.K_SCROLL) {
		stmt.ScrollOpt = ScrollYes
		p.next()
	}

	if p.isKw(scanner.K_FOR) {
		p.next()
		if p.isKw(scanner.K_EXECUTE) {
			p.next()
			query, endTok := p.readSQLUntilEndToken(scanner.K_USING, scanner.Token(';'), 0)
			stmt.DynQuery = query
			if endTok == scanner.K_USING {
				for {
					param, et := p.readSQLUntilEndToken(scanner.Token(','), scanner.Token(';'), 0)
					stmt.DynParams = append(stmt.DynParams, param)
					if et != scanner.Token(',') {
						break
					}
				}
			}
		} else {
			stmt.Query = p.readSQLUntilSemi()
			p.wantSelf(';')
		}
		return stmt
	}

	// Bound cursor with optional args
	if p.tok == scanner.Token('(') {
		args := p.readSQLUntil(scanner.Token(';'), 0, 0)
		stmt.CursorArgs = args
	}

	if p.tok == scanner.Token(';') {
		p.next()
	}

	return stmt
}

// parseStmtFetch parses: FETCH/MOVE [direction] cursor [INTO target] ;
func (p *Parser) parseStmtFetch(isMove bool) Stmt {
	pos := p.pos
	p.next() // consume FETCH or MOVE

	stmt := &StmtFetch{
		baseStmt: baseStmt{baseNode{Location: pos}},
		IsMove:   isMove,
	}

	// Parse optional direction
	dir := p.parseFetchDirection()
	stmt.Direction = dir

	// Cursor variable
	if p.tok == scanner.T_WORD || p.tok == scanner.IDENT || scanner.IsUnreservedKeyword(p.tok) {
		stmt.CurVar = p.lit
		p.next()
	}

	// INTO target (for FETCH only)
	if p.isKw(scanner.K_INTO) && !isMove {
		p.next()
		stmt.Target = p.readExprUntilSemi()
	}

	p.wantSelf(';')
	return stmt
}

// parseFetchDirection parses optional fetch direction keywords.
func (p *Parser) parseFetchDirection() string {
	switch p.tok {
	case scanner.K_NEXT, scanner.K_PRIOR, scanner.K_FIRST, scanner.K_LAST,
		scanner.K_ABSOLUTE, scanner.K_RELATIVE, scanner.K_FORWARD, scanner.K_BACKWARD:
		dir := p.lit
		p.next()
		// Some directions take a count
		if p.tok == scanner.ICONST {
			dir += " " + p.lit
			p.next()
		}
		// FROM or IN
		if p.isKw(scanner.K_FROM) || p.isKw(scanner.K_IN) {
			p.next()
		}
		return dir
	case scanner.K_ALL:
		p.next()
		if p.isKw(scanner.K_FROM) || p.isKw(scanner.K_IN) {
			p.next()
		}
		return "ALL"
	case scanner.ICONST:
		count := p.lit
		p.next()
		if p.isKw(scanner.K_FROM) || p.isKw(scanner.K_IN) {
			p.next()
		}
		return count
	}
	return ""
}

// parseStmtClose parses: CLOSE cursor ;
func (p *Parser) parseStmtClose() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_CLOSE)
	curVar := p.anyIdentifier()
	p.wantSelf(';')
	return &StmtClose{
		baseStmt: baseStmt{baseNode{Location: pos}},
		CurVar:   curVar,
	}
}

// parseStmtNull parses: NULL ;
func (p *Parser) parseStmtNull() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_NULL)
	p.wantSelf(';')
	return &StmtNull{baseStmt: baseStmt{baseNode{Location: pos}}}
}

// parseStmtCommit parses: COMMIT [AND [NO] CHAIN] ;
func (p *Parser) parseStmtCommit() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_COMMIT)
	chain := p.parseOptTransactionChain()
	p.wantSelf(';')
	return &StmtCommit{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Chain:    chain,
	}
}

// parseStmtRollback parses: ROLLBACK [AND [NO] CHAIN] ;
func (p *Parser) parseStmtRollback() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_ROLLBACK)
	chain := p.parseOptTransactionChain()
	p.wantSelf(';')
	return &StmtRollback{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Chain:    chain,
	}
}

// parseOptTransactionChain parses: [AND [NO] CHAIN]
func (p *Parser) parseOptTransactionChain() bool {
	if p.isKw(scanner.K_AND) {
		p.next()
		if p.isKw(scanner.K_NO) {
			p.next()
			p.wantKw(scanner.K_CHAIN)
			return false
		}
		p.wantKw(scanner.K_CHAIN)
		return true
	}
	return false
}

// parseExceptionSect parses: EXCEPTION WHEN cond THEN stmts [...]
func (p *Parser) parseExceptionSect() []*Exception {
	p.wantKw(scanner.K_EXCEPTION)
	var exceptions []*Exception
	for p.isKw(scanner.K_WHEN) {
		epos := p.pos
		p.next() // consume WHEN

		// Parse conditions: cond [OR cond ...]
		var conds []*Condition
		for {
			cond := p.parseCondition()
			conds = append(conds, cond)
			if !p.gotKw(scanner.K_OR) {
				break
			}
		}

		p.wantKw(scanner.K_THEN)
		body := p.parseProcSect()

		exceptions = append(exceptions, &Exception{
			baseNode:   baseNode{Location: epos},
			Conditions: conds,
			Body:       body,
		})
	}
	return exceptions
}

// parseCondition parses a single exception condition.
func (p *Parser) parseCondition() *Condition {
	name := p.anyIdentifier()
	if strings.ToLower(name) == "sqlstate" {
		if p.tok == scanner.SCONST {
			code := p.lit
			p.next()
			return &Condition{SQLState: code}
		}
	}
	return &Condition{Name: name}
}
