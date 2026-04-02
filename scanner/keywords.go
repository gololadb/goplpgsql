package scanner

import "strings"

type keywordEntry struct {
	token    Token
	category KeywordCategory
}

// plpgsqlKeywords maps lowercase keyword strings to their token and category.
var plpgsqlKeywords map[string]keywordEntry

func init() {
	plpgsqlKeywords = make(map[string]keywordEntry, 128)

	// Reserved keywords (from pl_reserved_kwlist.h)
	for _, e := range []struct {
		name  string
		token Token
	}{
		{"all", K_ALL},
		{"begin", K_BEGIN},
		{"by", K_BY},
		{"case", K_CASE},
		{"declare", K_DECLARE},
		{"else", K_ELSE},
		{"end", K_END},
		{"for", K_FOR},
		{"foreach", K_FOREACH},
		{"from", K_FROM},
		{"if", K_IF},
		{"in", K_IN},
		{"into", K_INTO},
		{"loop", K_LOOP},
		{"not", K_NOT},
		{"null", K_NULL},
		{"or", K_OR},
		{"then", K_THEN},
		{"to", K_TO},
		{"using", K_USING},
		{"when", K_WHEN},
		{"while", K_WHILE},
	} {
		plpgsqlKeywords[e.name] = keywordEntry{token: e.token, category: ReservedKeyword}
	}

	// Unreserved keywords (from pl_unreserved_kwlist.h)
	for _, e := range []struct {
		name  string
		token Token
	}{
		{"absolute", K_ABSOLUTE},
		{"alias", K_ALIAS},
		{"and", K_AND},
		{"array", K_ARRAY},
		{"assert", K_ASSERT},
		{"backward", K_BACKWARD},
		{"call", K_CALL},
		{"chain", K_CHAIN},
		{"close", K_CLOSE},
		{"collate", K_COLLATE},
		{"column", K_COLUMN},
		{"column_name", K_COLUMN_NAME},
		{"commit", K_COMMIT},
		{"constant", K_CONSTANT},
		{"constraint", K_CONSTRAINT},
		{"constraint_name", K_CONSTRAINT_NAME},
		{"continue", K_CONTINUE},
		{"current", K_CURRENT},
		{"cursor", K_CURSOR},
		{"datatype", K_DATATYPE},
		{"debug", K_DEBUG},
		{"default", K_DEFAULT},
		{"detail", K_DETAIL},
		{"diagnostics", K_DIAGNOSTICS},
		{"do", K_DO},
		{"dump", K_DUMP},
		{"elseif", K_ELSIF},
		{"elsif", K_ELSIF},
		{"errcode", K_ERRCODE},
		{"error", K_ERROR},
		{"exception", K_EXCEPTION},
		{"execute", K_EXECUTE},
		{"exit", K_EXIT},
		{"fetch", K_FETCH},
		{"first", K_FIRST},
		{"forward", K_FORWARD},
		{"get", K_GET},
		{"hint", K_HINT},
		{"import", K_IMPORT},
		{"info", K_INFO},
		{"insert", K_INSERT},
		{"is", K_IS},
		{"last", K_LAST},
		{"log", K_LOG},
		{"merge", K_MERGE},
		{"message", K_MESSAGE},
		{"message_text", K_MESSAGE_TEXT},
		{"move", K_MOVE},
		{"next", K_NEXT},
		{"no", K_NO},
		{"notice", K_NOTICE},
		{"open", K_OPEN},
		{"option", K_OPTION},
		{"perform", K_PERFORM},
		{"pg_context", K_PG_CONTEXT},
		{"pg_datatype_name", K_PG_DATATYPE_NAME},
		{"pg_exception_context", K_PG_EXCEPTION_CONTEXT},
		{"pg_exception_detail", K_PG_EXCEPTION_DETAIL},
		{"pg_exception_hint", K_PG_EXCEPTION_HINT},
		{"pg_routine_oid", K_PG_ROUTINE_OID},
		{"print_strict_params", K_PRINT_STRICT_PARAMS},
		{"prior", K_PRIOR},
		{"query", K_QUERY},
		{"raise", K_RAISE},
		{"relative", K_RELATIVE},
		{"return", K_RETURN},
		{"returned_sqlstate", K_RETURNED_SQLSTATE},
		{"reverse", K_REVERSE},
		{"rollback", K_ROLLBACK},
		{"row_count", K_ROW_COUNT},
		{"rowtype", K_ROWTYPE},
		{"schema", K_SCHEMA},
		{"schema_name", K_SCHEMA_NAME},
		{"scroll", K_SCROLL},
		{"slice", K_SLICE},
		{"sqlstate", K_SQLSTATE},
		{"stacked", K_STACKED},
		{"strict", K_STRICT},
		{"table", K_TABLE},
		{"table_name", K_TABLE_NAME},
		{"type", K_TYPE},
		{"use_column", K_USE_COLUMN},
		{"use_variable", K_USE_VARIABLE},
		{"variable_conflict", K_VARIABLE_CONFLICT},
		{"warning", K_WARNING},
	} {
		plpgsqlKeywords[e.name] = keywordEntry{token: e.token, category: UnreservedKeyword}
	}
}

// LookupKeyword returns the token and category for a PL/pgSQL keyword.
// The input is matched case-insensitively. Returns (0, 0, false) if not found.
func LookupKeyword(word string) (Token, KeywordCategory, bool) {
	e, ok := plpgsqlKeywords[strings.ToLower(word)]
	if !ok {
		return 0, 0, false
	}
	return e.token, e.category, true
}

// unreservedSet is populated at init time with all unreserved keyword tokens.
var unreservedSet map[Token]bool

func init() {
	unreservedSet = make(map[Token]bool)
	for _, e := range plpgsqlKeywords {
		if e.category == UnreservedKeyword {
			unreservedSet[e.token] = true
		}
	}
}

// IsUnreservedKeyword returns true if tok is an unreserved PL/pgSQL keyword.
func IsUnreservedKeyword(tok Token) bool {
	return unreservedSet[tok]
}
