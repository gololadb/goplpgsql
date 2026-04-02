package parser

import "github.com/gololadb/goplpgsql/scanner"

// parseStmtIf parses: IF expr THEN stmts [ELSIF ...] [ELSE stmts] END IF ;
func (p *Parser) parseStmtIf() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_IF)

	cond := p.readExprUntilThen()
	p.wantKw(scanner.K_THEN)

	thenBody := p.parseProcSect()

	var elsifs []*ElsIf
	for p.isKw(scanner.K_ELSIF) {
		epos := p.pos
		p.next()
		econd := p.readExprUntilThen()
		p.wantKw(scanner.K_THEN)
		ebody := p.parseProcSect()
		elsifs = append(elsifs, &ElsIf{
			baseNode:  baseNode{Location: epos},
			Condition: econd,
			Body:      ebody,
		})
	}

	var elseBody []Stmt
	if p.gotKw(scanner.K_ELSE) {
		elseBody = p.parseProcSect()
	}

	p.wantKw(scanner.K_END)
	p.wantKw(scanner.K_IF)
	p.wantSelf(';')

	return &StmtIf{
		baseStmt:  baseStmt{baseNode{Location: pos}},
		Condition: cond,
		ThenBody:  thenBody,
		ElsIfs:    elsifs,
		ElseBody:  elseBody,
	}
}

// parseStmtCase parses: CASE [expr] WHEN expr THEN stmts [...] [ELSE stmts] END CASE ;
func (p *Parser) parseStmtCase() Stmt {
	pos := p.pos
	p.wantKw(scanner.K_CASE)

	// Optional search expression (if next token is not WHEN)
	var searchExpr string
	if !p.isKw(scanner.K_WHEN) {
		searchExpr = p.readSQLUntil(scanner.K_WHEN, 0, 0)
	}

	var whens []*CaseWhen
	for p.isKw(scanner.K_WHEN) {
		wpos := p.pos
		p.next()
		wexpr := p.readExprUntilThen()
		p.wantKw(scanner.K_THEN)
		wbody := p.parseProcSect()
		whens = append(whens, &CaseWhen{
			baseNode: baseNode{Location: wpos},
			Expr:     wexpr,
			Body:     wbody,
		})
	}

	var elseBody []Stmt
	if p.gotKw(scanner.K_ELSE) {
		elseBody = p.parseProcSect()
	}

	p.wantKw(scanner.K_END)
	p.wantKw(scanner.K_CASE)
	p.wantSelf(';')

	return &StmtCase{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Expr:     searchExpr,
		Whens:    whens,
		ElseBody: elseBody,
	}
}

// parseStmtLoop parses: [<<label>>] LOOP stmts END LOOP [label] ;
func (p *Parser) parseStmtLoop(label string) Stmt {
	pos := p.pos
	p.wantKw(scanner.K_LOOP)

	body := p.parseLoopBody()

	return &StmtLoop{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Label:    label,
		Body:     body,
	}
}

// parseStmtWhile parses: [<<label>>] WHILE expr LOOP stmts END LOOP [label] ;
func (p *Parser) parseStmtWhile(label string) Stmt {
	pos := p.pos
	p.wantKw(scanner.K_WHILE)

	cond := p.readExprUntilLoop()
	p.wantKw(scanner.K_LOOP)

	body := p.parseLoopBody()

	return &StmtWhile{
		baseStmt:  baseStmt{baseNode{Location: pos}},
		Label:     label,
		Condition: cond,
		Body:      body,
	}
}

// parseStmtFor parses FOR loops (integer range, query, cursor, dynamic).
func (p *Parser) parseStmtFor(label string) Stmt {
	pos := p.pos
	p.wantKw(scanner.K_FOR)

	// Loop variable
	varName := p.anyIdentifier()

	p.wantKw(scanner.K_IN)

	// Check for REVERSE
	reverse := false
	if p.isKw(scanner.K_REVERSE) {
		reverse = true
		p.next()
	}

	// Check for EXECUTE (dynamic FOR)
	if p.isKw(scanner.K_EXECUTE) {
		p.next()
		query, endTok := p.readSQLUntilEndToken(scanner.K_LOOP, scanner.K_USING, 0)

		var params []string
		if endTok == scanner.K_USING {
			for {
				param, et := p.readSQLUntilEndToken(scanner.Token(','), scanner.K_LOOP, 0)
				params = append(params, param)
				if et != scanner.Token(',') {
					break
				}
			}
		}

		body := p.parseLoopBody()

		_ = params // dynamic params stored in query for simplicity
		return &StmtForS{
			baseStmt: baseStmt{baseNode{Location: pos}},
			Label:    label,
			Var:      varName,
			Query:    "EXECUTE " + query,
			Body:     body,
		}
	}

	// Read expression until .. or LOOP
	expr1, endTok := p.readSQLUntilEndToken(scanner.DOT_DOT, scanner.K_LOOP, 0)

	if endTok == scanner.DOT_DOT {
		// Integer FOR loop: FOR var IN [REVERSE] lower .. upper [BY step] LOOP
		upper, endTok2 := p.readSQLUntilEndToken(scanner.K_LOOP, scanner.K_BY, 0)

		var step string
		if endTok2 == scanner.K_BY {
			step = p.readSQLUntil(scanner.K_LOOP, 0, 0)
			p.wantKw(scanner.K_LOOP)
		}

		body := p.parseLoopBody()

		return &StmtForI{
			baseStmt: baseStmt{baseNode{Location: pos}},
			Label:    label,
			Var:      varName,
			Reverse:  reverse,
			Lower:    expr1,
			Upper:    upper,
			Step:     step,
			Body:     body,
		}
	}

	// Query FOR loop
	body := p.parseLoopBody()

	return &StmtForS{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Label:    label,
		Var:      varName,
		Query:    expr1,
		Body:     body,
	}
}

// parseStmtForEach parses: FOREACH var [SLICE n] IN ARRAY expr LOOP stmts END LOOP ;
func (p *Parser) parseStmtForEach(label string) Stmt {
	pos := p.pos
	p.wantKw(scanner.K_FOREACH)

	varName := p.anyIdentifier()

	slice := 0
	if p.isKw(scanner.K_SLICE) {
		p.next()
		if p.tok == scanner.ICONST {
			// parse integer
			for _, c := range p.lit {
				slice = slice*10 + int(c-'0')
			}
			p.next()
		}
	}

	p.wantKw(scanner.K_IN)
	p.wantKw(scanner.K_ARRAY)

	expr := p.readExprUntilLoop()
	p.wantKw(scanner.K_LOOP)

	body := p.parseLoopBody()

	return &StmtForEachA{
		baseStmt: baseStmt{baseNode{Location: pos}},
		Label:    label,
		Var:      varName,
		Slice:    slice,
		Expr:     expr,
		Body:     body,
	}
}

// parseLoopBody parses: stmts END LOOP [label] ;
func (p *Parser) parseLoopBody() []Stmt {
	body := p.parseProcSect()
	p.wantKw(scanner.K_END)
	p.wantKw(scanner.K_LOOP)
	// optional end label
	if p.tok == scanner.T_WORD || p.tok == scanner.IDENT || scanner.IsUnreservedKeyword(p.tok) {
		if p.tok != scanner.Token(';') {
			p.next()
		}
	}
	p.wantSelf(';')
	return body
}

// parseStmtExit parses: EXIT|CONTINUE [label] [WHEN expr] ;
func (p *Parser) parseStmtExit(isExit bool) Stmt {
	pos := p.pos
	p.next() // consume EXIT or CONTINUE

	var label string
	var cond string

	// Optional label (identifier that is not WHEN and not ;)
	if p.tok == scanner.T_WORD || p.tok == scanner.IDENT || scanner.IsUnreservedKeyword(p.tok) {
		label = p.lit
		p.next()
	}

	// Optional WHEN condition
	if p.isKw(scanner.K_WHEN) {
		p.next()
		cond = p.readExprUntilSemi()
	}

	p.wantSelf(';')

	return &StmtExit{
		baseStmt:  baseStmt{baseNode{Location: pos}},
		IsExit:    isExit,
		Label:     label,
		Condition: cond,
	}
}
