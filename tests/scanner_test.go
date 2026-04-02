package tests

import (
	"testing"

	"github.com/gololadb/goplpgsql/scanner"
)

func scanAll(src string) []scanner.Token {
	var s scanner.Scanner
	s.Init([]byte(src), nil)
	var toks []scanner.Token
	for {
		s.Next()
		toks = append(toks, s.Tok)
		if s.Tok == scanner.EOF {
			break
		}
	}
	return toks
}

func scanLits(src string) []string {
	var s scanner.Scanner
	s.Init([]byte(src), nil)
	var lits []string
	for {
		s.Next()
		if s.Tok == scanner.EOF {
			break
		}
		lits = append(lits, s.Lit)
	}
	return lits
}

func TestScannerKeywords(t *testing.T) {
	var s scanner.Scanner
	s.Init([]byte("BEGIN END IF THEN ELSE ELSIF LOOP WHILE FOR RETURN"), nil)

	expected := []scanner.Token{
		scanner.K_BEGIN, scanner.K_END, scanner.K_IF, scanner.K_THEN,
		scanner.K_ELSE, scanner.K_ELSIF, scanner.K_LOOP, scanner.K_WHILE,
		scanner.K_FOR, scanner.K_RETURN,
	}

	for i, exp := range expected {
		s.Next()
		if s.Tok != exp {
			t.Errorf("token %d: got %v, want %v (lit=%q)", i, s.Tok, exp, s.Lit)
		}
	}
}

func TestScannerIdentifiers(t *testing.T) {
	var s scanner.Scanner
	s.Init([]byte("my_var _foo bar123"), nil)

	for _, exp := range []string{"my_var", "_foo", "bar123"} {
		s.Next()
		if s.Tok != scanner.T_WORD {
			t.Errorf("expected T_WORD for %q, got %v", exp, s.Tok)
		}
		if s.Lit != exp {
			t.Errorf("expected lit %q, got %q", exp, s.Lit)
		}
	}
}

func TestScannerNumbers(t *testing.T) {
	var s scanner.Scanner
	s.Init([]byte("42 3.14 1e10 .5"), nil)

	s.Next()
	if s.Tok != scanner.ICONST || s.Lit != "42" {
		t.Errorf("expected ICONST 42, got %v %q", s.Tok, s.Lit)
	}
	s.Next()
	if s.Tok != scanner.FCONST || s.Lit != "3.14" {
		t.Errorf("expected FCONST 3.14, got %v %q", s.Tok, s.Lit)
	}
	s.Next()
	if s.Tok != scanner.FCONST || s.Lit != "1e10" {
		t.Errorf("expected FCONST 1e10, got %v %q", s.Tok, s.Lit)
	}
	s.Next()
	if s.Tok != scanner.FCONST || s.Lit != ".5" {
		t.Errorf("expected FCONST .5, got %v %q", s.Tok, s.Lit)
	}
}

func TestScannerStrings(t *testing.T) {
	var s scanner.Scanner
	s.Init([]byte("'hello' 'it''s' $$body$$ $tag$content$tag$"), nil)

	s.Next()
	if s.Tok != scanner.SCONST || s.Lit != "hello" {
		t.Errorf("expected SCONST hello, got %v %q", s.Tok, s.Lit)
	}
	s.Next()
	if s.Tok != scanner.SCONST || s.Lit != "it's" {
		t.Errorf("expected SCONST it's, got %v %q", s.Tok, s.Lit)
	}
	s.Next()
	if s.Tok != scanner.SCONST || s.Lit != "body" {
		t.Errorf("expected SCONST body, got %v %q", s.Tok, s.Lit)
	}
	s.Next()
	if s.Tok != scanner.SCONST || s.Lit != "content" {
		t.Errorf("expected SCONST content, got %v %q", s.Tok, s.Lit)
	}
}

func TestScannerOperators(t *testing.T) {
	var s scanner.Scanner
	s.Init([]byte(":= :: => << >> <= >= <> !="), nil)

	tests := []struct {
		tok scanner.Token
		lit string
	}{
		{scanner.COLON_EQUALS, ":="},
		{scanner.TYPECAST, "::"},
		{scanner.EQUALS_GREATER, "=>"},
		{scanner.LESS_LESS, "<<"},
		{scanner.GREATER_GREATER, ">>"},
		{scanner.LESS_EQUALS, "<="},
		{scanner.GREATER_EQUALS, ">="},
		{scanner.NOT_EQUALS, "<>"},
		{scanner.NOT_EQUALS, "!="},
	}

	for _, tt := range tests {
		s.Next()
		if s.Tok != tt.tok || s.Lit != tt.lit {
			t.Errorf("expected %v %q, got %v %q", tt.tok, tt.lit, s.Tok, s.Lit)
		}
	}
}

func TestScannerComments(t *testing.T) {
	lits := scanLits("BEGIN -- this is a comment\nEND")
	if len(lits) != 2 || lits[0] != "begin" || lits[1] != "end" {
		t.Errorf("expected [begin end], got %v", lits)
	}

	lits = scanLits("BEGIN /* block comment */ END")
	if len(lits) != 2 || lits[0] != "begin" || lits[1] != "end" {
		t.Errorf("expected [begin end], got %v", lits)
	}

	// Nested block comments
	lits = scanLits("BEGIN /* outer /* inner */ still comment */ END")
	if len(lits) != 2 || lits[0] != "begin" || lits[1] != "end" {
		t.Errorf("expected [begin end], got %v", lits)
	}
}

func TestScannerParams(t *testing.T) {
	var s scanner.Scanner
	s.Init([]byte("$1 $23"), nil)

	s.Next()
	if s.Tok != scanner.PARAM || s.Lit != "$1" {
		t.Errorf("expected PARAM $1, got %v %q", s.Tok, s.Lit)
	}
	s.Next()
	if s.Tok != scanner.PARAM || s.Lit != "$23" {
		t.Errorf("expected PARAM $23, got %v %q", s.Tok, s.Lit)
	}
}

func TestScannerCompositeIdent(t *testing.T) {
	var s scanner.Scanner
	s.Init([]byte("schema.table.column"), nil)

	s.Next()
	if s.Tok != scanner.T_CWORD || s.Lit != "schema.table.column" {
		t.Errorf("expected T_CWORD schema.table.column, got %v %q", s.Tok, s.Lit)
	}
}

func TestScannerQuotedIdent(t *testing.T) {
	var s scanner.Scanner
	s.Init([]byte(`"My Column"`), nil)

	s.Next()
	if s.Tok != scanner.IDENT || s.Lit != "My Column" {
		t.Errorf("expected IDENT 'My Column', got %v %q", s.Tok, s.Lit)
	}
}

func TestScannerPushBack(t *testing.T) {
	var s scanner.Scanner
	s.Init([]byte("BEGIN END"), nil)

	s.Next()
	if s.Tok != scanner.K_BEGIN {
		t.Fatalf("expected K_BEGIN")
	}
	s.PushBack()
	s.Next()
	if s.Tok != scanner.K_BEGIN {
		t.Fatalf("expected K_BEGIN after pushback, got %v", s.Tok)
	}
	s.Next()
	if s.Tok != scanner.K_END {
		t.Fatalf("expected K_END, got %v", s.Tok)
	}
}
