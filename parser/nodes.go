package parser

// Node is the interface implemented by all AST nodes.
type Node interface {
	Pos() int // byte offset in source, or -1 if unknown
	node()
}

// Stmt is the interface for statement nodes.
type Stmt interface {
	Node
	stmt()
}

// baseNode provides a default Pos implementation.
type baseNode struct {
	Location int
}

func (n *baseNode) Pos() int { return n.Location }
func (*baseNode) node()      {}

type baseStmt struct{ baseNode }

func (*baseStmt) stmt() {}

// ---------------------------------------------------------------------------
// Statement types (mirrors PLpgSQL_stmt_type from plpgsql.h)
// ---------------------------------------------------------------------------

// StmtBlock represents a DECLARE ... BEGIN ... END block.
type StmtBlock struct {
	baseStmt
	Label      string
	Body       []Stmt
	Decls      []*DeclVar
	Exceptions []*Exception
}

// DeclVar represents a variable declaration in a DECLARE section.
type DeclVar struct {
	baseNode
	Name       string
	DataType   string // raw type text
	Constant   bool
	NotNull    bool
	Default    string // raw default expression text
	IsAlias    bool
	AliasFor   string
	IsCursor   bool
	CursorArgs []*CursorArg
	CursorQuery string
	ScrollOpt  ScrollOption
}

func (*DeclVar) node() {}

// CursorArg represents a cursor argument declaration.
type CursorArg struct {
	Name     string
	DataType string
}

// ScrollOption for cursor declarations.
type ScrollOption int

const (
	ScrollDefault  ScrollOption = iota
	ScrollYes
	ScrollNo
)

// Exception represents a WHEN ... THEN block in an EXCEPTION section.
type Exception struct {
	baseNode
	Conditions []*Condition
	Body       []Stmt
}

func (*Exception) node() {}

// Condition represents a single exception condition (name or SQLSTATE).
type Condition struct {
	Name     string // condition name (e.g. "division_by_zero")
	SQLState string // 5-char SQLSTATE code, if specified via SQLSTATE 'xxxxx'
}

// StmtAssign represents variable := expression.
type StmtAssign struct {
	baseStmt
	Variable string // variable name (may be dotted)
	Expr     string // raw expression text
}

// StmtIf represents IF ... THEN ... ELSIF ... ELSE ... END IF.
type StmtIf struct {
	baseStmt
	Condition string // raw expression text
	ThenBody  []Stmt
	ElsIfs    []*ElsIf
	ElseBody  []Stmt
}

// ElsIf represents an ELSIF clause.
type ElsIf struct {
	baseNode
	Condition string
	Body      []Stmt
}

func (*ElsIf) node() {}

// StmtCase represents CASE ... WHEN ... END CASE.
type StmtCase struct {
	baseStmt
	Expr     string // search expression (empty for searched CASE)
	Whens    []*CaseWhen
	ElseBody []Stmt
}

// CaseWhen represents a WHEN clause in a CASE statement.
type CaseWhen struct {
	baseNode
	Expr string
	Body []Stmt
}

func (*CaseWhen) node() {}

// StmtLoop represents an unconditional LOOP ... END LOOP.
type StmtLoop struct {
	baseStmt
	Label string
	Body  []Stmt
}

// StmtWhile represents WHILE ... LOOP ... END LOOP.
type StmtWhile struct {
	baseStmt
	Label     string
	Condition string
	Body      []Stmt
}

// StmtForI represents FOR var IN lower .. upper [BY step] LOOP ... END LOOP.
type StmtForI struct {
	baseStmt
	Label   string
	Var     string
	Reverse bool
	Lower   string
	Upper   string
	Step    string // empty if no BY clause
	Body    []Stmt
}

// StmtForS represents FOR var IN query LOOP ... END LOOP.
type StmtForS struct {
	baseStmt
	Label string
	Var   string
	Query string
	Body  []Stmt
}

// StmtForC represents FOR var IN cursor LOOP ... END LOOP.
type StmtForC struct {
	baseStmt
	Label     string
	Var       string
	Cursor    string
	CursorArgs string
	Body      []Stmt
}

// StmtForEachA represents FOREACH var [SLICE n] IN ARRAY expr LOOP ... END LOOP.
type StmtForEachA struct {
	baseStmt
	Label string
	Var   string
	Slice int
	Expr  string
	Body  []Stmt
}

// StmtExit represents EXIT [label] [WHEN condition].
type StmtExit struct {
	baseStmt
	IsExit    bool // true for EXIT, false for CONTINUE
	Label     string
	Condition string
}

// StmtReturn represents RETURN [expression].
type StmtReturn struct {
	baseStmt
	Expr string
}

// StmtReturnNext represents RETURN NEXT expression.
type StmtReturnNext struct {
	baseStmt
	Expr string
}

// StmtReturnQuery represents RETURN QUERY query or RETURN QUERY EXECUTE expr.
type StmtReturnQuery struct {
	baseStmt
	Query      string
	DynQuery   string // non-empty if RETURN QUERY EXECUTE
	DynParams  []string
}

// StmtRaise represents RAISE [level] [format_string [, expr ...]] [USING ...].
type StmtRaise struct {
	baseStmt
	Level     string // DEBUG, LOG, INFO, NOTICE, WARNING, EXCEPTION, or ""
	Message   string // format string
	Params    []string
	Options   []*RaiseOption
	CondName  string // condition name or SQLSTATE
}

// RaiseOption represents a USING option in a RAISE statement.
type RaiseOption struct {
	OptType string // MESSAGE, DETAIL, HINT, ERRCODE, COLUMN, CONSTRAINT, DATATYPE, TABLE, SCHEMA
	Expr    string
}

// StmtAssert represents ASSERT condition [, message].
type StmtAssert struct {
	baseStmt
	Condition string
	Message   string
}

// StmtExecSQL represents an embedded SQL statement.
type StmtExecSQL struct {
	baseStmt
	SQL    string
	Into   bool
	Strict bool
	Target string
}

// StmtDynExecute represents EXECUTE string_expr [INTO ...] [USING ...].
type StmtDynExecute struct {
	baseStmt
	Query  string
	Into   bool
	Strict bool
	Target string
	Params []string
}

// StmtPerform represents PERFORM query.
type StmtPerform struct {
	baseStmt
	Expr string
}

// StmtCall represents CALL procedure(...) or DO block.
type StmtCall struct {
	baseStmt
	Expr   string
	IsCall bool // true for CALL, false for DO
}

// StmtGetDiag represents GET [CURRENT|STACKED] DIAGNOSTICS target = item [, ...].
type StmtGetDiag struct {
	baseStmt
	IsStacked bool
	Items     []*DiagItem
}

// DiagItem represents a single GET DIAGNOSTICS item.
type DiagItem struct {
	Target string
	Kind   string // ROW_COUNT, PG_CONTEXT, etc.
}

// StmtOpen represents OPEN cursor [...].
type StmtOpen struct {
	baseStmt
	CurVar    string
	ScrollOpt ScrollOption
	Query     string // for unbound cursors: the query
	DynQuery  string // for OPEN ... FOR EXECUTE
	DynParams []string
	CursorArgs string // for bound cursors: argument expressions
}

// StmtFetch represents FETCH [direction] cursor INTO target.
type StmtFetch struct {
	baseStmt
	Direction string
	CurVar    string
	Target    string
	IsMove    bool
}

// StmtClose represents CLOSE cursor.
type StmtClose struct {
	baseStmt
	CurVar string
}

// StmtNull represents NULL; (a no-op statement).
type StmtNull struct {
	baseStmt
}

// StmtCommit represents COMMIT [AND [NO] CHAIN].
type StmtCommit struct {
	baseStmt
	Chain bool
}

// StmtRollback represents ROLLBACK [AND [NO] CHAIN].
type StmtRollback struct {
	baseStmt
	Chain bool
}
