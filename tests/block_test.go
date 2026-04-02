package tests

import (
	"testing"

	"github.com/gololadb/goplpgsql/parser"
)

func TestMinimalBlock(t *testing.T) {
	block := parseOK(t, "BEGIN END")
	if len(block.Body) != 0 {
		t.Errorf("expected empty body, got %d stmts", len(block.Body))
	}
}

func TestMinimalBlockWithSemicolon(t *testing.T) {
	block := parseOK(t, "BEGIN END;")
	if len(block.Body) != 0 {
		t.Errorf("expected empty body, got %d stmts", len(block.Body))
	}
}

func TestBlockWithLabel(t *testing.T) {
	block := parseOK(t, "<<myblock>> BEGIN END myblock")
	if block.Label != "myblock" {
		t.Errorf("expected label 'myblock', got %q", block.Label)
	}
}

func TestBlockWithDeclare(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			x integer;
			y text;
		BEGIN
			NULL;
		END
	`)
	if len(block.Decls) != 2 {
		t.Fatalf("expected 2 decls, got %d", len(block.Decls))
	}
	if block.Decls[0].Name != "x" {
		t.Errorf("expected decl name 'x', got %q", block.Decls[0].Name)
	}
	if block.Decls[1].Name != "y" {
		t.Errorf("expected decl name 'y', got %q", block.Decls[1].Name)
	}
}

func TestDeclConstant(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			pi CONSTANT numeric := 3.14159;
		BEGIN
			NULL;
		END
	`)
	if len(block.Decls) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(block.Decls))
	}
	d := block.Decls[0]
	if !d.Constant {
		t.Error("expected constant")
	}
	if d.Default != "3.14159" {
		t.Errorf("expected default '3.14159', got %q", d.Default)
	}
}

func TestDeclNotNull(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			x integer NOT NULL := 0;
		BEGIN
			NULL;
		END
	`)
	d := block.Decls[0]
	if !d.NotNull {
		t.Error("expected NOT NULL")
	}
	if d.Default != "0" {
		t.Errorf("expected default '0', got %q", d.Default)
	}
}

func TestDeclDefault(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			x integer DEFAULT 42;
			y text := 'hello';
			z integer = 0;
		BEGIN
			NULL;
		END
	`)
	if block.Decls[0].Default != "42" {
		t.Errorf("expected default '42', got %q", block.Decls[0].Default)
	}
	// SCONST strips quotes, so the literal is just "hello"
	if block.Decls[1].Default != "hello" {
		t.Errorf("expected default \"hello\", got %q", block.Decls[1].Default)
	}
	if block.Decls[2].Default != "0" {
		t.Errorf("expected default '0', got %q", block.Decls[2].Default)
	}
}

func TestDeclAlias(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			x ALIAS FOR $1;
		BEGIN
			NULL;
		END
	`)
	d := block.Decls[0]
	if !d.IsAlias {
		t.Error("expected alias")
	}
	if d.AliasFor != "$1" {
		t.Errorf("expected alias for '$1', got %q", d.AliasFor)
	}
}

func TestDeclCursor(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c CURSOR FOR SELECT * FROM t;
		BEGIN
			NULL;
		END
	`)
	d := block.Decls[0]
	if !d.IsCursor {
		t.Error("expected cursor")
	}
	// Keywords are lowercased by the scanner
	if d.CursorQuery != "SELECT * from t" {
		t.Errorf("expected cursor query 'SELECT * from t', got %q", d.CursorQuery)
	}
}

func TestDeclCursorWithArgs(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c CURSOR (key integer) FOR SELECT * FROM t WHERE id = key;
		BEGIN
			NULL;
		END
	`)
	d := block.Decls[0]
	if !d.IsCursor {
		t.Error("expected cursor")
	}
	if len(d.CursorArgs) != 1 {
		t.Fatalf("expected 1 cursor arg, got %d", len(d.CursorArgs))
	}
	if d.CursorArgs[0].Name != "key" {
		t.Errorf("expected arg name 'key', got %q", d.CursorArgs[0].Name)
	}
}

func TestDeclScrollCursor(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c SCROLL CURSOR FOR SELECT 1;
			d NO SCROLL CURSOR FOR SELECT 2;
		BEGIN
			NULL;
		END
	`)
	if block.Decls[0].ScrollOpt != parser.ScrollYes {
		t.Error("expected SCROLL")
	}
	if block.Decls[1].ScrollOpt != parser.ScrollNo {
		t.Error("expected NO SCROLL")
	}
}

func TestNestedBlock(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			BEGIN
				NULL;
			END;
		END
	`)
	if len(block.Body) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(block.Body))
	}
	inner, ok := block.Body[0].(*parser.StmtBlock)
	if !ok {
		t.Fatalf("expected StmtBlock, got %T", block.Body[0])
	}
	if len(inner.Body) != 1 {
		t.Errorf("expected 1 inner stmt, got %d", len(inner.Body))
	}
}

func TestLabeledNestedBlock(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			<<inner>>
			DECLARE
				x integer;
			BEGIN
				NULL;
			END inner;
		END
	`)
	if len(block.Body) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(block.Body))
	}
	inner, ok := block.Body[0].(*parser.StmtBlock)
	if !ok {
		t.Fatalf("expected StmtBlock, got %T", block.Body[0])
	}
	if inner.Label != "inner" {
		t.Errorf("expected label 'inner', got %q", inner.Label)
	}
}

func TestExceptionBlock(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			NULL;
		EXCEPTION
			WHEN division_by_zero THEN
				NULL;
			WHEN others THEN
				NULL;
		END
	`)
	if len(block.Exceptions) != 2 {
		t.Fatalf("expected 2 exceptions, got %d", len(block.Exceptions))
	}
	if block.Exceptions[0].Conditions[0].Name != "division_by_zero" {
		t.Errorf("expected condition 'division_by_zero', got %q", block.Exceptions[0].Conditions[0].Name)
	}
}

func TestExceptionSQLState(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			NULL;
		EXCEPTION
			WHEN sqlstate '22012' THEN
				NULL;
		END
	`)
	if len(block.Exceptions) != 1 {
		t.Fatalf("expected 1 exception, got %d", len(block.Exceptions))
	}
	if block.Exceptions[0].Conditions[0].SQLState != "22012" {
		t.Errorf("expected SQLSTATE '22012', got %q", block.Exceptions[0].Conditions[0].SQLState)
	}
}

func TestExceptionOrConditions(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			NULL;
		EXCEPTION
			WHEN division_by_zero OR unique_violation THEN
				NULL;
		END
	`)
	exc := block.Exceptions[0]
	if len(exc.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(exc.Conditions))
	}
}
