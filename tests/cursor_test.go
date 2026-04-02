package tests

import (
	"testing"

	"github.com/gololadb/goplpgsql/parser"
)

func TestOpenCursorForQuery(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c refcursor;
		BEGIN
			OPEN c FOR SELECT * FROM t;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtOpen)
	if !ok {
		t.Fatalf("expected StmtOpen, got %T", block.Body[0])
	}
	if s.CurVar != "c" {
		t.Errorf("expected curvar 'c', got %q", s.CurVar)
	}
	if s.Query != "SELECT * from t" {
		t.Errorf("expected query, got %q", s.Query)
	}
}

func TestOpenCursorForExecute(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c refcursor;
		BEGIN
			OPEN c FOR EXECUTE 'SELECT 1';
		END
	`)
	s := block.Body[0].(*parser.StmtOpen)
	// SCONST strips quotes
	if s.DynQuery != "SELECT 1" {
		t.Errorf("expected dyn query, got %q", s.DynQuery)
	}
}

func TestOpenCursorScroll(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c refcursor;
		BEGIN
			OPEN c SCROLL FOR SELECT 1;
		END
	`)
	s := block.Body[0].(*parser.StmtOpen)
	if s.ScrollOpt != parser.ScrollYes {
		t.Error("expected SCROLL")
	}
}

func TestOpenCursorNoScroll(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c refcursor;
		BEGIN
			OPEN c NO SCROLL FOR SELECT 1;
		END
	`)
	s := block.Body[0].(*parser.StmtOpen)
	if s.ScrollOpt != parser.ScrollNo {
		t.Error("expected NO SCROLL")
	}
}

func TestFetchSimple(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c refcursor;
			r record;
		BEGIN
			FETCH c INTO r;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtFetch)
	if !ok {
		t.Fatalf("expected StmtFetch, got %T", block.Body[0])
	}
	if s.CurVar != "c" {
		t.Errorf("expected curvar 'c', got %q", s.CurVar)
	}
	if s.IsMove {
		t.Error("expected IsMove=false")
	}
}

func TestFetchNext(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c refcursor;
			r record;
		BEGIN
			FETCH NEXT FROM c INTO r;
		END
	`)
	s := block.Body[0].(*parser.StmtFetch)
	if s.Direction != "next" {
		t.Errorf("expected direction 'next', got %q", s.Direction)
	}
}

func TestFetchFirst(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c refcursor;
			r record;
		BEGIN
			FETCH FIRST FROM c INTO r;
		END
	`)
	s := block.Body[0].(*parser.StmtFetch)
	if s.Direction != "first" {
		t.Errorf("expected direction 'first', got %q", s.Direction)
	}
}

func TestMoveSimple(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c refcursor;
		BEGIN
			MOVE c;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtFetch)
	if !ok {
		t.Fatalf("expected StmtFetch, got %T", block.Body[0])
	}
	if !s.IsMove {
		t.Error("expected IsMove=true")
	}
}

func TestCloseSimple(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c refcursor;
		BEGIN
			CLOSE c;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtClose)
	if !ok {
		t.Fatalf("expected StmtClose, got %T", block.Body[0])
	}
	if s.CurVar != "c" {
		t.Errorf("expected curvar 'c', got %q", s.CurVar)
	}
}
