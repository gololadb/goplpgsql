package tests

import (
	"testing"

	"github.com/gololadb/goplpgsql/parser"
)

// Realistic PL/pgSQL function bodies from common patterns.

func TestComplexFunction(t *testing.T) {
	// A function that uses most PL/pgSQL features
	block := parseOK(t, `
		<<main>>
		DECLARE
			v_count integer := 0;
			v_name text;
			v_rec record;
			c CURSOR FOR SELECT id, name FROM users WHERE active;
		BEGIN
			-- Simple assignment
			v_count := 0;

			-- IF/ELSIF/ELSE
			IF v_count = 0 THEN
				RAISE NOTICE 'count is zero';
			ELSIF v_count < 10 THEN
				RAISE NOTICE 'count is %', v_count;
			ELSE
				RAISE WARNING 'count is high: %', v_count;
			END IF;

			-- FOR loop with integer range
			FOR i IN 1 .. 10 LOOP
				v_count := v_count + i;
			END LOOP;

			-- FOR loop with query
			FOR v_rec IN SELECT * FROM users LOOP
				v_name := v_rec.name;
			END LOOP;

			-- WHILE loop
			WHILE v_count > 0 LOOP
				v_count := v_count - 1;
				EXIT WHEN v_count = 5;
				CONTINUE WHEN v_count = 7;
			END LOOP;

			-- CASE
			CASE v_count
				WHEN 0 THEN
					NULL;
				WHEN 1 THEN
					NULL;
				ELSE
					RAISE EXCEPTION 'unexpected count: %', v_count;
			END CASE;

			-- Dynamic SQL
			EXECUTE 'SELECT count(*) FROM users' INTO STRICT v_count;

			-- PERFORM
			PERFORM pg_sleep(1);

			-- RETURN
			RETURN v_count;
		EXCEPTION
			WHEN division_by_zero THEN
				RAISE NOTICE 'division by zero caught';
				RETURN -1;
			WHEN others THEN
				GET STACKED DIAGNOSTICS v_name = MESSAGE_TEXT;
				RAISE NOTICE 'error: %', v_name;
				RETURN -2;
		END main
	`)

	if block.Label != "main" {
		t.Errorf("expected label 'main', got %q", block.Label)
	}
	if len(block.Decls) != 4 {
		t.Errorf("expected 4 decls, got %d", len(block.Decls))
	}
	if len(block.Exceptions) != 2 {
		t.Errorf("expected 2 exceptions, got %d", len(block.Exceptions))
	}
	// Body should have: assign, if, for_i, for_s, while, case, execute, perform, return = 9
	if len(block.Body) != 9 {
		t.Errorf("expected 9 body stmts, got %d", len(block.Body))
	}
}

func TestTriggerFunction(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			v_old_val text;
			v_new_val text;
		BEGIN
			IF TG_OP = 'INSERT' THEN
				INSERT INTO audit_log (op, new_data)
					VALUES ('INSERT', row_to_json(NEW));
				RETURN NEW;
			ELSIF TG_OP = 'UPDATE' THEN
				INSERT INTO audit_log (op, old_data, new_data)
					VALUES ('UPDATE', row_to_json(OLD), row_to_json(NEW));
				RETURN NEW;
			ELSIF TG_OP = 'DELETE' THEN
				INSERT INTO audit_log (op, old_data)
					VALUES ('DELETE', row_to_json(OLD));
				RETURN OLD;
			END IF;
			RETURN NULL;
		END
	`)
	if len(block.Decls) != 2 {
		t.Errorf("expected 2 decls, got %d", len(block.Decls))
	}
	// IF + RETURN NULL = 2
	if len(block.Body) != 2 {
		t.Errorf("expected 2 body stmts, got %d", len(block.Body))
	}
	ifStmt := block.Body[0].(*parser.StmtIf)
	if len(ifStmt.ElsIfs) != 2 {
		t.Errorf("expected 2 elsifs, got %d", len(ifStmt.ElsIfs))
	}
}

func TestCursorManipulation(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			c1 CURSOR FOR SELECT * FROM employees;
			c2 refcursor;
			emp record;
		BEGIN
			OPEN c1;
			OPEN c2 SCROLL FOR SELECT * FROM departments;

			FETCH NEXT FROM c1 INTO emp;
			WHILE FOUND LOOP
				RAISE NOTICE 'employee: %', emp.name;
				FETCH NEXT FROM c1 INTO emp;
			END LOOP;

			MOVE FORWARD 5 FROM c2;
			FETCH c2 INTO emp;

			CLOSE c1;
			CLOSE c2;
		END
	`)
	if len(block.Decls) != 3 {
		t.Errorf("expected 3 decls, got %d", len(block.Decls))
	}
	// OPEN, OPEN, FETCH, WHILE, MOVE, FETCH, CLOSE, CLOSE = 8
	if len(block.Body) != 8 {
		t.Errorf("expected 8 body stmts, got %d", len(block.Body))
	}
}

func TestReturnTableFunction(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			r record;
		BEGIN
			FOR r IN
				SELECT id, name, salary
				FROM employees
				WHERE department_id = dept_id
				ORDER BY salary DESC
			LOOP
				IF r.salary > threshold THEN
					RETURN NEXT r;
				END IF;
			END LOOP;
			RETURN;
		END
	`)
	// FOR + RETURN = 2
	if len(block.Body) != 2 {
		t.Errorf("expected 2 body stmts, got %d", len(block.Body))
	}
}

func TestDynamicSQLFunction(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			v_sql text;
			v_result integer;
			v_table text := 'users';
		BEGIN
			v_sql := 'SELECT count(*) FROM ' || v_table;
			EXECUTE v_sql INTO v_result;

			EXECUTE 'INSERT INTO log (msg) VALUES ($1)'
				USING 'count is ' || v_result;

			EXECUTE 'SELECT $1 + $2'
				INTO v_result
				USING 10, 20;

			RETURN v_result;
		END
	`)
	// assign, execute, execute, execute, return = 5
	if len(block.Body) != 5 {
		t.Errorf("expected 5 body stmts, got %d", len(block.Body))
	}
}

func TestNestedBlocks(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			x integer := 0;
		BEGIN
			<<inner1>>
			BEGIN
				x := x + 1;
			EXCEPTION
				WHEN others THEN
					NULL;
			END inner1;

			<<inner2>>
			DECLARE
				y integer := 10;
			BEGIN
				x := x + y;
			END inner2;

			RETURN x;
		END
	`)
	// inner1 block, inner2 block, return = 3
	if len(block.Body) != 3 {
		t.Errorf("expected 3 body stmts, got %d", len(block.Body))
	}
	inner1 := block.Body[0].(*parser.StmtBlock)
	if inner1.Label != "inner1" {
		t.Errorf("expected label 'inner1', got %q", inner1.Label)
	}
	if len(inner1.Exceptions) != 1 {
		t.Errorf("expected 1 exception in inner1, got %d", len(inner1.Exceptions))
	}
	inner2 := block.Body[1].(*parser.StmtBlock)
	if inner2.Label != "inner2" {
		t.Errorf("expected label 'inner2', got %q", inner2.Label)
	}
	if len(inner2.Decls) != 1 {
		t.Errorf("expected 1 decl in inner2, got %d", len(inner2.Decls))
	}
}

func TestForEachArrayFunction(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			arr integer[] := ARRAY[1, 2, 3, 4, 5];
			elem integer;
			total integer := 0;
		BEGIN
			FOREACH elem IN ARRAY arr LOOP
				total := total + elem;
			END LOOP;
			ASSERT total = 15, 'sum should be 15';
			RETURN total;
		END
	`)
	// FOREACH, ASSERT, RETURN = 3
	if len(block.Body) != 3 {
		t.Errorf("expected 3 body stmts, got %d", len(block.Body))
	}
}

func TestCommitRollbackFunction(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			i integer;
		BEGIN
			FOR i IN 1 .. 100 LOOP
				INSERT INTO batch_log VALUES (i);
				IF i % 10 = 0 THEN
					COMMIT;
				END IF;
			END LOOP;
		EXCEPTION
			WHEN others THEN
				ROLLBACK;
				RAISE;
		END
	`)
	if len(block.Body) != 1 {
		t.Errorf("expected 1 body stmt (FOR), got %d", len(block.Body))
	}
	if len(block.Exceptions) != 1 {
		t.Errorf("expected 1 exception, got %d", len(block.Exceptions))
	}
}

func TestRaiseVariants(t *testing.T) {
	block := parseOK(t, `
		BEGIN
			RAISE DEBUG 'debug msg';
			RAISE LOG 'log msg';
			RAISE INFO 'info msg';
			RAISE NOTICE 'notice: % and %', a, b;
			RAISE WARNING 'warn msg';
			RAISE EXCEPTION 'err msg'
				USING ERRCODE = '22000',
					  HINT = 'check input',
					  DETAIL = 'bad value';
			RAISE SQLSTATE '22012';
			RAISE division_by_zero;
			RAISE;
		END
	`)
	if len(block.Body) != 9 {
		t.Errorf("expected 9 raise stmts, got %d", len(block.Body))
	}
	// Check the USING variant
	r := block.Body[5].(*parser.StmtRaise)
	if len(r.Options) != 3 {
		t.Errorf("expected 3 USING options, got %d", len(r.Options))
	}
}

func TestSearchedCase(t *testing.T) {
	block := parseOK(t, `
		DECLARE
			grade char(1);
			result text;
		BEGIN
			CASE
				WHEN grade = 'A' THEN
					result := 'Excellent';
				WHEN grade = 'B' THEN
					result := 'Good';
				WHEN grade = 'C' THEN
					result := 'Average';
				ELSE
					result := 'Unknown';
			END CASE;
			RETURN result;
		END
	`)
	if len(block.Body) != 2 {
		t.Errorf("expected 2 body stmts, got %d", len(block.Body))
	}
	c := block.Body[0].(*parser.StmtCase)
	if c.Expr != "" {
		t.Errorf("expected empty search expr for searched CASE, got %q", c.Expr)
	}
	if len(c.Whens) != 3 {
		t.Errorf("expected 3 WHEN clauses, got %d", len(c.Whens))
	}
	if len(c.ElseBody) != 1 {
		t.Errorf("expected 1 ELSE stmt, got %d", len(c.ElseBody))
	}
}
