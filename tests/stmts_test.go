package tests

import (
	"testing"

	"github.com/gololadb/goplpgsql/parser"
)

func TestAssignSimple(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			x := 42;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtAssign)
	if !ok {
		t.Fatalf("expected StmtAssign, got %T", block.Body[0])
	}
	if s.Variable != "x" {
		t.Errorf("expected var 'x', got %q", s.Variable)
	}
	if s.Expr != "42" {
		t.Errorf("expected expr '42', got %q", s.Expr)
	}
}

func TestAssignEquals(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			x = 42;
		END
	`)
	s := block.Body[0].(*parser.StmtAssign)
	if s.Variable != "x" || s.Expr != "42" {
		t.Errorf("expected x = 42, got %q = %q", s.Variable, s.Expr)
	}
}

func TestAssignExpression(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			x := y + z * 2;
		END
	`)
	s := block.Body[0].(*parser.StmtAssign)
	if s.Expr != "y + z * 2" {
		t.Errorf("expected expr 'y + z * 2', got %q", s.Expr)
	}
}

func TestReturnSimple(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RETURN;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtReturn)
	if !ok {
		t.Fatalf("expected StmtReturn, got %T", block.Body[0])
	}
	if s.Expr != "" {
		t.Errorf("expected empty expr, got %q", s.Expr)
	}
}

func TestReturnExpr(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RETURN x + 1;
		END
	`)
	s := block.Body[0].(*parser.StmtReturn)
	if s.Expr != "x + 1" {
		t.Errorf("expected expr 'x + 1', got %q", s.Expr)
	}
}

func TestReturnNext(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RETURN NEXT x;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtReturnNext)
	if !ok {
		t.Fatalf("expected StmtReturnNext, got %T", block.Body[0])
	}
	if s.Expr != "x" {
		t.Errorf("expected expr 'x', got %q", s.Expr)
	}
}

func TestReturnQuery(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RETURN QUERY SELECT * FROM t;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtReturnQuery)
	if !ok {
		t.Fatalf("expected StmtReturnQuery, got %T", block.Body[0])
	}
	if s.Query != "SELECT * from t" {
		t.Errorf("expected query 'SELECT * from t', got %q", s.Query)
	}
}

func TestReturnQueryExecute(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RETURN QUERY EXECUTE 'SELECT * FROM ' || tbl;
		END
	`)
	s := block.Body[0].(*parser.StmtReturnQuery)
	if s.DynQuery == "" {
		t.Error("expected dynamic query")
	}
}

func TestPerform(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			PERFORM my_func(1, 2);
		END
	`)
	s, ok := block.Body[0].(*parser.StmtPerform)
	if !ok {
		t.Fatalf("expected StmtPerform, got %T", block.Body[0])
	}
	if s.Expr != "my_func ( 1 , 2 )" {
		t.Errorf("unexpected expr %q", s.Expr)
	}
}

func TestCall(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			CALL my_proc(1);
		END
	`)
	s, ok := block.Body[0].(*parser.StmtCall)
	if !ok {
		t.Fatalf("expected StmtCall, got %T", block.Body[0])
	}
	if !s.IsCall {
		t.Error("expected IsCall=true")
	}
}

func TestDo(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			DO $$BEGIN NULL; END$$;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtCall)
	if !ok {
		t.Fatalf("expected StmtCall, got %T", block.Body[0])
	}
	if s.IsCall {
		t.Error("expected IsCall=false for DO")
	}
}

func TestNullStmt(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			NULL;
		END
	`)
	_, ok := block.Body[0].(*parser.StmtNull)
	if !ok {
		t.Fatalf("expected StmtNull, got %T", block.Body[0])
	}
}

func TestExecSQL(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			INSERT INTO t VALUES (1, 2);
		END
	`)
	s, ok := block.Body[0].(*parser.StmtExecSQL)
	if !ok {
		t.Fatalf("expected StmtExecSQL, got %T", block.Body[0])
	}
	// Keywords are lowercased by scanner
	if s.SQL != "insert into t VALUES ( 1 , 2 )" {
		t.Errorf("unexpected SQL %q", s.SQL)
	}
}

func TestDynExecute(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			EXECUTE 'SELECT 1';
		END
	`)
	s, ok := block.Body[0].(*parser.StmtDynExecute)
	if !ok {
		t.Fatalf("expected StmtDynExecute, got %T", block.Body[0])
	}
	// SCONST strips quotes
	if s.Query != "SELECT 1" {
		t.Errorf("expected query, got %q", s.Query)
	}
}

func TestDynExecuteInto(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			EXECUTE 'SELECT 1' INTO x;
		END
	`)
	s := block.Body[0].(*parser.StmtDynExecute)
	if !s.Into {
		t.Error("expected Into=true")
	}
	if s.Target != "x" {
		t.Errorf("expected target 'x', got %q", s.Target)
	}
}

func TestDynExecuteUsing(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			EXECUTE 'SELECT $1' USING x;
		END
	`)
	s := block.Body[0].(*parser.StmtDynExecute)
	if len(s.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(s.Params))
	}
}

func TestDynExecuteIntoStrict(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			EXECUTE 'SELECT 1' INTO STRICT x;
		END
	`)
	s := block.Body[0].(*parser.StmtDynExecute)
	if !s.Strict {
		t.Error("expected Strict=true")
	}
}

func TestCommit(t *testing.T) {
	block := parseOK(t, `BEGIN COMMIT; END`)
	_, ok := block.Body[0].(*parser.StmtCommit)
	if !ok {
		t.Fatalf("expected StmtCommit, got %T", block.Body[0])
	}
}

func TestCommitAndChain(t *testing.T) {
	block := parseOK(t, `BEGIN COMMIT AND CHAIN; END`)
	s := block.Body[0].(*parser.StmtCommit)
	if !s.Chain {
		t.Error("expected Chain=true")
	}
}

func TestRollback(t *testing.T) {
	block := parseOK(t, `BEGIN ROLLBACK; END`)
	_, ok := block.Body[0].(*parser.StmtRollback)
	if !ok {
		t.Fatalf("expected StmtRollback, got %T", block.Body[0])
	}
}

func TestRollbackAndNoChain(t *testing.T) {
	block := parseOK(t, `BEGIN ROLLBACK AND NO CHAIN; END`)
	s := block.Body[0].(*parser.StmtRollback)
	if s.Chain {
		t.Error("expected Chain=false")
	}
}

func TestGetDiagnostics(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			GET DIAGNOSTICS cnt = ROW_COUNT;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtGetDiag)
	if !ok {
		t.Fatalf("expected StmtGetDiag, got %T", block.Body[0])
	}
	if s.IsStacked {
		t.Error("expected IsStacked=false")
	}
	if len(s.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(s.Items))
	}
	if s.Items[0].Target != "cnt" {
		t.Errorf("expected target 'cnt', got %q", s.Items[0].Target)
	}
	if s.Items[0].Kind != "ROW_COUNT" {
		t.Errorf("expected kind 'ROW_COUNT', got %q", s.Items[0].Kind)
	}
}

func TestGetStackedDiagnostics(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			NULL;
		EXCEPTION WHEN others THEN
			GET STACKED DIAGNOSTICS v_sqlstate = RETURNED_SQLSTATE, v_msg = MESSAGE_TEXT;
		END
	`)
	exc := block.Exceptions[0]
	s, ok := exc.Body[0].(*parser.StmtGetDiag)
	if !ok {
		t.Fatalf("expected StmtGetDiag, got %T", exc.Body[0])
	}
	if !s.IsStacked {
		t.Error("expected IsStacked=true")
	}
	if len(s.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(s.Items))
	}
}
