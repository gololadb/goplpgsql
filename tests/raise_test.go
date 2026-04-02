package tests

import (
	"testing"

	"github.com/gololadb/goplpgsql/parser"
)

func TestRaiseSimple(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtRaise)
	if !ok {
		t.Fatalf("expected StmtRaise, got %T", block.Body[0])
	}
	if s.Level != "" || s.Message != "" {
		t.Errorf("expected bare RAISE, got level=%q msg=%q", s.Level, s.Message)
	}
}

func TestRaiseNotice(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE NOTICE 'hello %', name;
		END
	`)
	s := block.Body[0].(*parser.StmtRaise)
	if s.Level != "NOTICE" {
		t.Errorf("expected level NOTICE, got %q", s.Level)
	}
	if s.Message != "hello %" {
		t.Errorf("expected message 'hello %%', got %q", s.Message)
	}
	if len(s.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(s.Params))
	}
}

func TestRaiseException(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE EXCEPTION 'error: %', msg;
		END
	`)
	s := block.Body[0].(*parser.StmtRaise)
	if s.Level != "EXCEPTION" {
		t.Errorf("expected level EXCEPTION, got %q", s.Level)
	}
}

func TestRaiseWarning(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE WARNING 'watch out';
		END
	`)
	s := block.Body[0].(*parser.StmtRaise)
	if s.Level != "WARNING" {
		t.Errorf("expected level WARNING, got %q", s.Level)
	}
}

func TestRaiseDebug(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE DEBUG 'debug info';
		END
	`)
	s := block.Body[0].(*parser.StmtRaise)
	if s.Level != "DEBUG" {
		t.Errorf("expected level DEBUG, got %q", s.Level)
	}
}

func TestRaiseLog(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE LOG 'log entry';
		END
	`)
	s := block.Body[0].(*parser.StmtRaise)
	if s.Level != "LOG" {
		t.Errorf("expected level LOG, got %q", s.Level)
	}
}

func TestRaiseInfo(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE INFO 'info msg';
		END
	`)
	s := block.Body[0].(*parser.StmtRaise)
	if s.Level != "INFO" {
		t.Errorf("expected level INFO, got %q", s.Level)
	}
}

func TestRaiseUsing(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE EXCEPTION 'error' USING ERRCODE = '22000', HINT = 'try again';
		END
	`)
	s := block.Body[0].(*parser.StmtRaise)
	if len(s.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(s.Options))
	}
	if s.Options[0].OptType != "ERRCODE" {
		t.Errorf("expected ERRCODE, got %q", s.Options[0].OptType)
	}
	if s.Options[1].OptType != "HINT" {
		t.Errorf("expected HINT, got %q", s.Options[1].OptType)
	}
}

func TestRaiseSQLState(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE SQLSTATE '22012';
		END
	`)
	s := block.Body[0].(*parser.StmtRaise)
	if s.CondName != "22012" {
		t.Errorf("expected condname '22012', got %q", s.CondName)
	}
}

func TestRaiseConditionName(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE division_by_zero;
		END
	`)
	s := block.Body[0].(*parser.StmtRaise)
	if s.CondName != "division_by_zero" {
		t.Errorf("expected condname 'division_by_zero', got %q", s.CondName)
	}
}

func TestAssert(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			ASSERT x > 0;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtAssert)
	if !ok {
		t.Fatalf("expected StmtAssert, got %T", block.Body[0])
	}
	if s.Condition != "x > 0" {
		t.Errorf("expected condition 'x > 0', got %q", s.Condition)
	}
}

func TestAssertWithMessage(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			ASSERT x > 0, 'x must be positive';
		END
	`)
	s := block.Body[0].(*parser.StmtAssert)
	// String literals are re-quoted.
	if s.Message != "'x must be positive'" {
		t.Errorf("expected message, got %q", s.Message)
	}
}
