package tests

import (
	"testing"

	"github.com/gololadb/goplpgsql/parser"
)

func parseOK(t *testing.T, src string) *parser.StmtBlock {
	t.Helper()
	block, err := parser.Parse([]byte(src), nil)
	if err != nil {
		t.Fatalf("parse error: %v\n\nsource:\n%s", err, src)
	}
	if block == nil {
		t.Fatal("nil block returned")
	}
	return block
}

func parseFail(t *testing.T, src string) {
	t.Helper()
	_, err := parser.Parse([]byte(src), nil)
	if err == nil {
		t.Fatalf("expected parse error for:\n%s", src)
	}
}
