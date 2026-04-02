<div align="center">

  [![Build with Ona](https://ona.com/build-with-ona.svg)](https://app.ona.com/#https://github.com/gololadb/goplpgsql)

# goplpgsql

A PL/pgSQL parser in pure Go. No generated code, no grammar files, no flex/bison — just a recursive-descent scanner and parser modeled after PostgreSQL's `pl_scanner.c` and `pl_gram.y`.

</div>

Produces an AST that mirrors PostgreSQL's internal PL/pgSQL parse tree (`plpgsql.h`).

## Usage

```go
package main

import (
	"fmt"

	"github.com/gololadb/goplpgsql/parser"
)

func main() {
	src := []byte(`
DECLARE
    total integer := 0;
BEGIN
    FOR i IN 1..10 LOOP
        total := total + i;
    END LOOP;
    RAISE NOTICE 'Sum is %', total;
END;
`)

	block, err := parser.Parse(src, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Block has %d declarations and %d statements\n",
		len(block.Decls), len(block.Body))

	for _, stmt := range block.Body {
		switch s := stmt.(type) {
		case *parser.StmtForI:
			fmt.Printf("FOR %s IN %s..%s\n", s.Var, s.Lower, s.Upper)
		case *parser.StmtRaise:
			fmt.Printf("RAISE %s '%s'\n", s.Level, s.Message)
		}
	}
}
```

## API

The public API is a single function:

```go
func parser.Parse(src []byte, errh func(pos int, msg string)) (*parser.StmtBlock, error)
```

- `src` — PL/pgSQL function body as bytes
- `errh` — optional error handler called for each parse error (nil to collect only the first error)
- Returns the top-level `StmtBlock` representing the DECLARE/BEGIN/END block

All AST node types are exported from the `parser` package. Statements implement `parser.Stmt` and all nodes implement `parser.Node`.

## What's supported

### Statements (~30 types)

| Category | Statements |
|---|---|
| **Blocks** | DECLARE ... BEGIN ... END, nested labeled blocks |
| **Declarations** | Variable declarations (with DEFAULT, CONSTANT, NOT NULL), ALIAS FOR, cursor declarations (with arguments, SCROLL options) |
| **Assignment** | variable := expression |
| **Conditionals** | IF ... THEN ... ELSIF ... ELSE ... END IF, CASE (simple and searched) |
| **Loops** | LOOP, WHILE, FOR (integer range), FOR (query), FOR (cursor), FOREACH ... IN ARRAY (with SLICE) |
| **Loop control** | EXIT, CONTINUE (with optional label and WHEN condition) |
| **Return** | RETURN, RETURN NEXT, RETURN QUERY, RETURN QUERY EXECUTE ... USING |
| **RAISE / ASSERT** | RAISE (all levels: DEBUG through EXCEPTION, with format params and USING options), ASSERT |
| **SQL execution** | Embedded SQL (with INTO [STRICT] target), EXECUTE ... INTO ... USING, PERFORM |
| **Procedures** | CALL, DO |
| **Diagnostics** | GET [CURRENT\|STACKED] DIAGNOSTICS |
| **Cursors** | OPEN (bound, unbound, dynamic with EXECUTE/USING), FETCH, MOVE, CLOSE |
| **Transactions** | COMMIT, ROLLBACK (with AND [NO] CHAIN) |
| **Other** | NULL (no-op) |

### Exception handling

Full EXCEPTION block support with WHEN clauses, condition names, SQLSTATE codes, and OR-combined conditions.

### Declarations

- Typed variables with optional DEFAULT, CONSTANT, NOT NULL
- `%TYPE` and `%ROWTYPE` references (captured in the type text)
- ALIAS FOR parameter references
- Cursor declarations with arguments and SCROLL/NO SCROLL options

## Project structure

```
goplpgsql/
├── scanner/          # Lexical scanner (tokens, keywords, PL/pgSQL-specific scanning)
├── parser/           # Recursive-descent parser and AST node definitions
└── tests/            # Tests organized by PL/pgSQL feature
```

### Test organization

Tests are in `tests/` and organized by feature — each file covers a specific area of PL/pgSQL syntax:

```
tests/
├── scanner_test.go      # Lexical scanning: tokens, keywords, strings, operators
├── block_test.go        # DECLARE/BEGIN/END blocks, declarations, nested blocks
├── control_test.go      # IF, CASE, LOOP, WHILE, FOR, FOREACH, EXIT, CONTINUE
├── stmts_test.go        # Assignment, RETURN, EXECUTE, PERFORM, CALL, GET DIAGNOSTICS
├── cursor_test.go       # OPEN, FETCH, MOVE, CLOSE, cursor declarations
├── raise_test.go        # RAISE (all levels, format params, USING options), ASSERT
├── integration_test.go  # Full function bodies combining multiple features
└── helpers_test.go      # Test utilities
```

## Running tests

```bash
go test ./...
```

## Design

The parser follows the same architecture as [gopgsql](https://github.com/gololadb/gopgsql), adapted for PL/pgSQL's procedural grammar:

- **Scanner** (`scanner/`) reads UTF-8 source one rune at a time, producing tokens. Handles PL/pgSQL-specific lexical elements: the full keyword table from `pl_scanner.c`, composite identifiers (`schema.table.column`), dollar-quoted strings, the `:=` assignment operator, and `<<label>>` delimiters.
- **Parser** (`parser/`) is a recursive-descent parser where each grammar production is a Go function. SQL expressions and embedded SQL statements are captured as raw text (matching PostgreSQL's own approach of deferring SQL parsing to the main SQL parser).

### Why raw text for SQL expressions?

PL/pgSQL is a host language for SQL. PostgreSQL itself does not parse SQL expressions inside PL/pgSQL at function-definition time — it defers that to execution. This parser follows the same strategy: SQL fragments (expressions, queries, type names) are captured as strings, keeping the parser focused on PL/pgSQL control flow. Use [gopgsql](https://github.com/gololadb/gopgsql) to parse the SQL fragments if needed.

## Limitations

- Error recovery is minimal — the parser stops at the first error
- SQL expressions are captured as raw text, not parsed into AST nodes
- No semantic analysis (variable resolution, type checking)
- Position tracking uses a synthetic encoding, not raw byte offsets

## License

MIT
