package tests

import (
	"testing"

	"github.com/gololadb/goplpgsql/parser"
)

func TestIfSimple(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			IF x > 0 THEN
				NULL;
			END IF;
		END
	`)
	if len(block.Body) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(block.Body))
	}
	s, ok := block.Body[0].(*parser.StmtIf)
	if !ok {
		t.Fatalf("expected StmtIf, got %T", block.Body[0])
	}
	if s.Condition != "x > 0" {
		t.Errorf("expected condition 'x > 0', got %q", s.Condition)
	}
}

func TestIfElse(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			IF x > 0 THEN
				NULL;
			ELSE
				NULL;
			END IF;
		END
	`)
	s := block.Body[0].(*parser.StmtIf)
	if len(s.ElseBody) == 0 {
		t.Error("expected ELSE body")
	}
}

func TestIfElsif(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			IF x > 0 THEN
				NULL;
			ELSIF x = 0 THEN
				NULL;
			ELSIF x < 0 THEN
				NULL;
			ELSE
				NULL;
			END IF;
		END
	`)
	s := block.Body[0].(*parser.StmtIf)
	if len(s.ElsIfs) != 2 {
		t.Fatalf("expected 2 elsifs, got %d", len(s.ElsIfs))
	}
	if s.ElsIfs[0].Condition != "x = 0" {
		t.Errorf("expected elsif condition 'x = 0', got %q", s.ElsIfs[0].Condition)
	}
}

func TestCaseSearched(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			CASE
				WHEN x = 1 THEN
					NULL;
				WHEN x = 2 THEN
					NULL;
				ELSE
					NULL;
			END CASE;
		END
	`)
	s := block.Body[0].(*parser.StmtCase)
	if s.Expr != "" {
		t.Errorf("expected empty search expr, got %q", s.Expr)
	}
	if len(s.Whens) != 2 {
		t.Fatalf("expected 2 whens, got %d", len(s.Whens))
	}
}

func TestCaseSimple(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			CASE x
				WHEN 1 THEN
					NULL;
				WHEN 2 THEN
					NULL;
			END CASE;
		END
	`)
	s := block.Body[0].(*parser.StmtCase)
	if s.Expr != "x" {
		t.Errorf("expected search expr 'x', got %q", s.Expr)
	}
}

func TestSimpleLoop(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			LOOP
				EXIT;
			END LOOP;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtLoop)
	if !ok {
		t.Fatalf("expected StmtLoop, got %T", block.Body[0])
	}
	if len(s.Body) != 1 {
		t.Errorf("expected 1 body stmt, got %d", len(s.Body))
	}
}

func TestLabeledLoop(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			<<myloop>>
			LOOP
				EXIT myloop;
			END LOOP myloop;
		END
	`)
	s := block.Body[0].(*parser.StmtLoop)
	if s.Label != "myloop" {
		t.Errorf("expected label 'myloop', got %q", s.Label)
	}
}

func TestWhileLoop(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			WHILE i < 10 LOOP
				i := i + 1;
			END LOOP;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtWhile)
	if !ok {
		t.Fatalf("expected StmtWhile, got %T", block.Body[0])
	}
	if s.Condition != "i < 10" {
		t.Errorf("expected condition 'i < 10', got %q", s.Condition)
	}
}

func TestForIntegerLoop(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			FOR i IN 1 .. 10 LOOP
				NULL;
			END LOOP;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtForI)
	if !ok {
		t.Fatalf("expected StmtForI, got %T", block.Body[0])
	}
	if s.Var != "i" {
		t.Errorf("expected var 'i', got %q", s.Var)
	}
	if s.Lower != "1" {
		t.Errorf("expected lower '1', got %q", s.Lower)
	}
	if s.Upper != "10" {
		t.Errorf("expected upper '10', got %q", s.Upper)
	}
}

func TestForIntegerReverseByLoop(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			FOR i IN REVERSE 10 .. 1 BY 2 LOOP
				NULL;
			END LOOP;
		END
	`)
	s := block.Body[0].(*parser.StmtForI)
	if !s.Reverse {
		t.Error("expected REVERSE")
	}
	if s.Step != "2" {
		t.Errorf("expected step '2', got %q", s.Step)
	}
}

func TestForQueryLoop(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			FOR rec IN SELECT * FROM t LOOP
				NULL;
			END LOOP;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtForS)
	if !ok {
		t.Fatalf("expected StmtForS, got %T", block.Body[0])
	}
	if s.Var != "rec" {
		t.Errorf("expected var 'rec', got %q", s.Var)
	}
}

func TestForEachArray(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			FOREACH x IN ARRAY arr LOOP
				NULL;
			END LOOP;
		END
	`)
	s, ok := block.Body[0].(*parser.StmtForEachA)
	if !ok {
		t.Fatalf("expected StmtForEachA, got %T", block.Body[0])
	}
	if s.Var != "x" {
		t.Errorf("expected var 'x', got %q", s.Var)
	}
}

func TestForEachSlice(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			FOREACH x SLICE 1 IN ARRAY arr LOOP
				NULL;
			END LOOP;
		END
	`)
	s := block.Body[0].(*parser.StmtForEachA)
	if s.Slice != 1 {
		t.Errorf("expected slice 1, got %d", s.Slice)
	}
}

func TestExitSimple(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			LOOP
				EXIT;
			END LOOP;
		END
	`)
	loop := block.Body[0].(*parser.StmtLoop)
	s, ok := loop.Body[0].(*parser.StmtExit)
	if !ok {
		t.Fatalf("expected StmtExit, got %T", loop.Body[0])
	}
	if !s.IsExit {
		t.Error("expected IsExit=true")
	}
}

func TestExitWithLabel(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			<<outer>>
			LOOP
				EXIT outer;
			END LOOP;
		END
	`)
	loop := block.Body[0].(*parser.StmtLoop)
	s := loop.Body[0].(*parser.StmtExit)
	if s.Label != "outer" {
		t.Errorf("expected label 'outer', got %q", s.Label)
	}
}

func TestExitWhen(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			LOOP
				EXIT WHEN i > 10;
			END LOOP;
		END
	`)
	loop := block.Body[0].(*parser.StmtLoop)
	s := loop.Body[0].(*parser.StmtExit)
	if s.Condition != "i > 10" {
		t.Errorf("expected condition 'i > 10', got %q", s.Condition)
	}
}

func TestContinue(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			LOOP
				CONTINUE WHEN i = 5;
			END LOOP;
		END
	`)
	loop := block.Body[0].(*parser.StmtLoop)
	s := loop.Body[0].(*parser.StmtExit)
	if s.IsExit {
		t.Error("expected IsExit=false for CONTINUE")
	}
}
