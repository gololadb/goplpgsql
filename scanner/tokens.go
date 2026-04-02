// Package scanner implements a hand-written lexical scanner for PL/pgSQL,
// modeled after the gopgsql scanner architecture but targeting the lexical
// grammar defined in src/pl/plpgsql/src/pl_scanner.c of PostgreSQL.
package scanner

// Token represents a lexical token type.
type Token uint

const (
	_ Token = iota

	EOF // end of input

	// Identifiers and literals (inherited from core SQL scanner)
	IDENT  // identifier
	FCONST // floating-point constant
	SCONST // string constant
	ICONST // integer constant
	PARAM  // positional parameter ($1, $2, ...)
	Op     // operator

	// Multi-character fixed tokens
	TYPECAST       // ::
	DOT_DOT        // ..
	COLON_EQUALS   // :=
	EQUALS_GREATER // =>
	LESS_EQUALS    // <=
	GREATER_EQUALS // >=
	NOT_EQUALS     // != or <>
	LESS_LESS      // <<
	GREATER_GREATER // >>

	// PL/pgSQL-specific word tokens
	T_WORD  // unrecognized simple identifier
	T_CWORD // unrecognized composite identifier (a.b.c)
	T_DATUM // a known variable reference (not used in pure parsing, treated as T_WORD)
)

// Keyword tokens start at 256 to avoid collision with ASCII single-char tokens.
const (
	K_ALL Token = iota + 256
	K_BEGIN
	K_BY
	K_CASE
	K_DECLARE
	K_ELSE
	K_END
	K_FOR
	K_FOREACH
	K_FROM
	K_IF
	K_IN
	K_INTO
	K_LOOP
	K_NOT
	K_NULL
	K_OR
	K_THEN
	K_TO
	K_USING
	K_WHEN
	K_WHILE

	// ---- PL/pgSQL unreserved keywords ----
	K_ABSOLUTE
	K_ALIAS
	K_AND
	K_ARRAY
	K_ASSERT
	K_BACKWARD
	K_CALL
	K_CHAIN
	K_CLOSE
	K_COLLATE
	K_COLUMN
	K_COLUMN_NAME
	K_COMMIT
	K_CONSTANT
	K_CONSTRAINT
	K_CONSTRAINT_NAME
	K_CONTINUE
	K_CURRENT
	K_CURSOR
	K_DATATYPE
	K_DEBUG
	K_DEFAULT
	K_DETAIL
	K_DIAGNOSTICS
	K_DO
	K_DUMP
	K_ELSIF
	K_ERRCODE
	K_ERROR
	K_EXCEPTION
	K_EXECUTE
	K_EXIT
	K_FETCH
	K_FIRST
	K_FORWARD
	K_GET
	K_HINT
	K_IMPORT
	K_INFO
	K_INSERT
	K_IS
	K_LAST
	K_LOG
	K_MERGE
	K_MESSAGE
	K_MESSAGE_TEXT
	K_MOVE
	K_NEXT
	K_NO
	K_NOTICE
	K_OPEN
	K_OPTION
	K_PERFORM
	K_PG_CONTEXT
	K_PG_DATATYPE_NAME
	K_PG_EXCEPTION_CONTEXT
	K_PG_EXCEPTION_DETAIL
	K_PG_EXCEPTION_HINT
	K_PG_ROUTINE_OID
	K_PRINT_STRICT_PARAMS
	K_PRIOR
	K_QUERY
	K_RAISE
	K_RELATIVE
	K_RETURN
	K_RETURNED_SQLSTATE
	K_REVERSE
	K_ROLLBACK
	K_ROW_COUNT
	K_ROWTYPE
	K_SCHEMA
	K_SCHEMA_NAME
	K_SCROLL
	K_SLICE
	K_SQLSTATE
	K_STACKED
	K_STRICT
	K_TABLE
	K_TABLE_NAME
	K_TYPE
	K_USE_COLUMN
	K_USE_VARIABLE
	K_VARIABLE_CONFLICT
	K_WARNING

	tokenCount
)

// KeywordCategory classifies PL/pgSQL keywords.
type KeywordCategory uint8

const (
	ReservedKeyword   KeywordCategory = iota // cannot be used as identifier
	UnreservedKeyword                        // can be used as identifier
)
