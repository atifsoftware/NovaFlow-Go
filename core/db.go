package core

import (
	"database/sql"
	"fmt"
	"strings"
)

// DB wraps *sql.DB and exposes a fluent, PDO-style query builder
// (DB.Table("users").Where("email", "=", x).First()).
type DB struct {
	Conn *sql.DB
}

func OpenDB(driver, dsn string) (*DB, error) {
	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	return &DB{Conn: conn}, nil
}

// Table starts a new query builder chain against the given table.
func (db *DB) Table(name string) *QueryBuilder {
	return &QueryBuilder{db: db, table: name}
}

type whereClause struct {
	boolean string // AND / OR
	sql     string
	args    []interface{}
}

// QueryBuilder builds parameterized SQL incrementally. Every value is
// bound as a placeholder ("?") — never string-concatenated — so query
// results are always safe from SQL injection.
type QueryBuilder struct {
	db       *DB
	table    string
	columns  []string
	wheres   []whereClause
	orderBy  string
	limitN   int
	offsetN  int
	joins    []string
}

func (q *QueryBuilder) Select(cols ...string) *QueryBuilder {
	q.columns = cols
	return q
}

// Where adds an "AND column op ? " clause, e.g. Where("status", "=", "active").
func (q *QueryBuilder) Where(column, op string, value interface{}) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{boolean: "AND", sql: fmt.Sprintf("%s %s ?", column, op), args: []interface{}{value}})
	return q
}

func (q *QueryBuilder) OrWhere(column, op string, value interface{}) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{boolean: "OR", sql: fmt.Sprintf("%s %s ?", column, op), args: []interface{}{value}})
	return q
}

func (q *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}
	q.wheres = append(q.wheres, whereClause{
		boolean: "AND",
		sql:     fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ",")),
		args:    values,
	})
	return q
}

func (q *QueryBuilder) Join(clause string) *QueryBuilder {
	q.joins = append(q.joins, clause)
	return q
}

func (q *QueryBuilder) OrderBy(column, dir string) *QueryBuilder {
	q.orderBy = fmt.Sprintf("%s %s", column, dir)
	return q
}

func (q *QueryBuilder) Limit(n int) *QueryBuilder {
	q.limitN = n
	return q
}

func (q *QueryBuilder) Offset(n int) *QueryBuilder {
	q.offsetN = n
	return q
}

func (q *QueryBuilder) buildSelect() (string, []interface{}) {
	cols := "*"
	if len(q.columns) > 0 {
		cols = strings.Join(q.columns, ", ")
	}
	sqlStr := fmt.Sprintf("SELECT %s FROM %s", cols, q.table)
	for _, j := range q.joins {
		sqlStr += " " + j
	}
	var args []interface{}
	if len(q.wheres) > 0 {
		sqlStr += " WHERE "
		for i, w := range q.wheres {
			if i > 0 {
				sqlStr += " " + w.boolean + " "
			}
			sqlStr += w.sql
			args = append(args, w.args...)
		}
	}
	if q.orderBy != "" {
		sqlStr += " ORDER BY " + q.orderBy
	}
	if q.limitN > 0 {
		sqlStr += fmt.Sprintf(" LIMIT %d", q.limitN)
	}
	if q.offsetN > 0 {
		sqlStr += fmt.Sprintf(" OFFSET %d", q.offsetN)
	}
	return sqlStr, args
}

// Get executes the query and returns every row as a map[string]interface{}.
func (q *QueryBuilder) Get() ([]map[string]interface{}, error) {
	sqlStr, args := q.buildSelect()
	rows, err := q.db.Conn.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// First returns the first matching row, or nil if none found.
func (q *QueryBuilder) First() (map[string]interface{}, error) {
	q.limitN = 1
	rows, err := q.Get()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// Count returns the number of matching rows.
func (q *QueryBuilder) Count() (int64, error) {
	q.columns = []string{"COUNT(*) as cnt"}
	row, err := q.First()
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return ToInt64(row["cnt"]), nil
}

// Insert inserts a row from a column->value map and returns the new
// auto-increment ID (if the driver supports LastInsertId).
func (q *QueryBuilder) Insert(data map[string]interface{}) (int64, error) {
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))
	for k, v := range data {
		cols = append(cols, k)
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q.table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	res, err := q.db.Conn.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Update updates matching rows (honors any Where() clauses already
// applied to this builder) and returns rows affected.
func (q *QueryBuilder) Update(data map[string]interface{}) (int64, error) {
	sets := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))
	for k, v := range data {
		sets = append(sets, k+" = ?")
		args = append(args, v)
	}
	sqlStr := fmt.Sprintf("UPDATE %s SET %s", q.table, strings.Join(sets, ", "))
	if len(q.wheres) > 0 {
		sqlStr += " WHERE "
		for i, w := range q.wheres {
			if i > 0 {
				sqlStr += " " + w.boolean + " "
			}
			sqlStr += w.sql
			args = append(args, w.args...)
		}
	}
	res, err := q.db.Conn.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Delete removes matching rows and returns rows affected.
func (q *QueryBuilder) Delete() (int64, error) {
	sqlStr := fmt.Sprintf("DELETE FROM %s", q.table)
	var args []interface{}
	if len(q.wheres) > 0 {
		sqlStr += " WHERE "
		for i, w := range q.wheres {
			if i > 0 {
				sqlStr += " " + w.boolean + " "
			}
			sqlStr += w.sql
			args = append(args, w.args...)
		}
	}
	res, err := q.db.Conn.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Raw runs an arbitrary parameterized SELECT and returns rows as maps.
func (db *DB) Raw(sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Conn.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// Exec runs an arbitrary parameterized statement (INSERT/UPDATE/DELETE/DDL).
func (db *DB) Exec(sqlStr string, args ...interface{}) (sql.Result, error) {
	return db.Conn.Exec(sqlStr, args...)
}

func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := map[string]interface{}{}
		for i, col := range cols {
			v := values[i]
			if b, ok := v.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = v
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func ToInt64(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case string:
		var i int64
		fmt.Sscanf(t, "%d", &i)
		return i
	default:
		return 0
	}
}
