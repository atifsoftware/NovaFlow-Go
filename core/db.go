package core

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// SQLQueryer defines common database operations shared by *sql.DB and *sql.Tx.
type SQLQueryer interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Dialect abstracts database-specific SQL syntax differences across MySQL, PostgreSQL, and SQLite.
type Dialect interface {
	DriverName() string
	Placeholder(index int) string
	QuoteIdentifier(name string) string
	SupportsLastInsertID() bool
	BuildDSN(cfg *Config) string
}

// MySQLDialect implements Dialect for MySQL/MariaDB.
type MySQLDialect struct{}

func (d *MySQLDialect) DriverName() string                 { return "mysql" }
func (d *MySQLDialect) Placeholder(index int) string       { return "?" }
func (d *MySQLDialect) QuoteIdentifier(name string) string { return "`" + name + "`" }
func (d *MySQLDialect) SupportsLastInsertID() bool        { return true }
func (d *MySQLDialect) BuildDSN(cfg *Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		cfg.Get("DB_USER", "root"),
		cfg.Get("DB_PASS", ""),
		cfg.Get("DB_HOST", "127.0.0.1"),
		cfg.Get("DB_PORT", "3306"),
		cfg.Get("DB_NAME", "novaflow_db"),
	)
}

// PostgresDialect implements Dialect for PostgreSQL.
type PostgresDialect struct{}

func (d *PostgresDialect) DriverName() string                 { return "postgres" }
func (d *PostgresDialect) Placeholder(index int) string       { return fmt.Sprintf("$%d", index) }
func (d *PostgresDialect) QuoteIdentifier(name string) string { return `"` + name + `"` }
func (d *PostgresDialect) SupportsLastInsertID() bool        { return false }
func (d *PostgresDialect) BuildDSN(cfg *Config) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Get("DB_USER", "postgres"),
		cfg.Get("DB_PASS", ""),
		cfg.Get("DB_HOST", "127.0.0.1"),
		cfg.Get("DB_PORT", "5432"),
		cfg.Get("DB_NAME", "novaflow_db"),
		cfg.Get("DB_SSLMODE", "disable"),
	)
}

// SQLiteDialect implements Dialect for SQLite using pure-Go glebarez driver.
type SQLiteDialect struct{}

func (d *SQLiteDialect) DriverName() string                 { return "sqlite" }
func (d *SQLiteDialect) Placeholder(index int) string       { return "?" }
func (d *SQLiteDialect) QuoteIdentifier(name string) string { return `"` + name + `"` }
func (d *SQLiteDialect) SupportsLastInsertID() bool        { return true }
func (d *SQLiteDialect) BuildDSN(cfg *Config) string {
	path := cfg.Get("DB_DATABASE", "storage/database.sqlite")
	if path == "" {
		path = "storage/database.sqlite"
	}
	return path
}

// GetDialect returns the Dialect corresponding to the driver name.
func GetDialect(driver string) Dialect {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "postgres", "postgresql", "pq", "pgx":
		return &PostgresDialect{}
	case "sqlite", "sqlite3":
		return &SQLiteDialect{}
	default:
		return &MySQLDialect{}
	}
}

// DB wraps *sql.DB and exposes a fluent, multi-dialect query builder.
type DB struct {
	Conn    *sql.DB
	Dialect Dialect
}

// OpenDB opens a database connection with auto-detected dialect from the driver name.
func OpenDB(driver, dsn string) (*DB, error) {
	return OpenDBWithDialect(GetDialect(driver), dsn)
}

// OpenDBWithDialect opens a connection using a specific Dialect.
func OpenDBWithDialect(dialect Dialect, dsn string) (*DB, error) {
	conn, err := sql.Open(dialect.DriverName(), dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	// Optimize connection pooling for production load
	conn.SetMaxOpenConns(100)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(time.Hour)

	return &DB{Conn: conn, Dialect: dialect}, nil
}

// Begin starts a new database transaction.
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.Conn.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, dialect: db.Dialect}, nil
}

// Table starts a new query builder chain against the given table using the connection.
func (db *DB) Table(name string) *QueryBuilder {
	return &QueryBuilder{
		db:      db,
		queryer: db.Conn,
		dialect: db.Dialect,
		table:   name,
	}
}

// Tx wraps *sql.Tx and provides fluent QueryBuilder access.
type Tx struct {
	tx      *sql.Tx
	dialect Dialect
}

// Commit commits the active transaction.
func (tx *Tx) Commit() error {
	return tx.tx.Commit()
}

// Rollback rolls back the active transaction.
func (tx *Tx) Rollback() error {
	return tx.tx.Rollback()
}

// Table starts a new query builder chain against the given table within the transaction.
func (tx *Tx) Table(name string) *QueryBuilder {
	return &QueryBuilder{
		db:      nil,
		queryer: tx.tx,
		dialect: tx.dialect,
		table:   name,
	}
}

type whereClause struct {
	boolean  string // AND / OR
	column   string
	op       string
	value    interface{}
	inValues []interface{}
	isRaw    bool
	rawSQL   string
	rawArgs  []interface{}
}

// QueryBuilder builds dialect-safe parameterized SQL incrementally.
type QueryBuilder struct {
	db      *DB
	queryer SQLQueryer
	dialect Dialect
	table   string
	columns []string
	wheres  []whereClause
	orderBy string
	limitN  int
	offsetN int
	joins   []string
}

func (q *QueryBuilder) getDialect() Dialect {
	if q.dialect != nil {
		return q.dialect
	}
	if q.db != nil && q.db.Dialect != nil {
		return q.db.Dialect
	}
	return &MySQLDialect{}
}

func (q *QueryBuilder) Select(cols ...string) *QueryBuilder {
	q.columns = cols
	return q
}

// Where adds an "AND column op value" clause.
func (q *QueryBuilder) Where(column, op string, value interface{}) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{
		boolean: "AND",
		column:  column,
		op:      op,
		value:   value,
	})
	return q
}

// OrWhere adds an "OR column op value" clause.
func (q *QueryBuilder) OrWhere(column, op string, value interface{}) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{
		boolean: "OR",
		column:  column,
		op:      op,
		value:   value,
	})
	return q
}

// WhereIn adds an "AND column IN (...)" clause.
func (q *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
	q.wheres = append(q.wheres, whereClause{
		boolean:  "AND",
		column:   column,
		inValues: values,
	})
	return q
}

// Join appends a raw join clause.
func (q *QueryBuilder) Join(clause string) *QueryBuilder {
	q.joins = append(q.joins, clause)
	return q
}

// InnerJoin appends an INNER JOIN clause to the query.
func (q *QueryBuilder) InnerJoin(table, first, op, second string) *QueryBuilder {
	q.joins = append(q.joins, fmt.Sprintf("INNER JOIN %s ON %s %s %s", table, first, op, second))
	return q
}

// LeftJoin appends a LEFT JOIN clause to the query.
func (q *QueryBuilder) LeftJoin(table, first, op, second string) *QueryBuilder {
	q.joins = append(q.joins, fmt.Sprintf("LEFT JOIN %s ON %s %s %s", table, first, op, second))
	return q
}

// RightJoin appends a RIGHT JOIN clause to the query.
func (q *QueryBuilder) RightJoin(table, first, op, second string) *QueryBuilder {
	q.joins = append(q.joins, fmt.Sprintf("RIGHT JOIN %s ON %s %s %s", table, first, op, second))
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

// Clone creates a deep copy of the QueryBuilder state.
func (q *QueryBuilder) Clone() *QueryBuilder {
	clone := &QueryBuilder{
		db:      q.db,
		queryer: q.queryer,
		dialect: q.dialect,
		table:   q.table,
		orderBy: q.orderBy,
		limitN:  q.limitN,
		offsetN: q.offsetN,
	}
	if len(q.columns) > 0 {
		clone.columns = make([]string, len(q.columns))
		copy(clone.columns, q.columns)
	}
	if len(q.wheres) > 0 {
		clone.wheres = make([]whereClause, len(q.wheres))
		copy(clone.wheres, q.wheres)
	}
	if len(q.joins) > 0 {
		clone.joins = make([]string, len(q.joins))
		copy(clone.joins, q.joins)
	}
	return clone
}

func (q *QueryBuilder) buildWhere(startParamIdx int) (string, []interface{}) {
	if len(q.wheres) == 0 {
		return "", nil
	}

	var sb strings.Builder
	var args []interface{}
	paramIdx := startParamIdx
	dialect := q.getDialect()

	for i, w := range q.wheres {
		if i > 0 {
			sb.WriteString(" " + w.boolean + " ")
		}

		if len(w.inValues) > 0 {
			placeholders := make([]string, len(w.inValues))
			for j := range w.inValues {
				placeholders[j] = dialect.Placeholder(paramIdx)
				paramIdx++
			}
			sb.WriteString(fmt.Sprintf("%s IN (%s)", w.column, strings.Join(placeholders, ", ")))
			args = append(args, w.inValues...)
		} else if w.isRaw {
			sb.WriteString(w.rawSQL)
			args = append(args, w.rawArgs...)
		} else {
			sb.WriteString(fmt.Sprintf("%s %s %s", w.column, w.op, dialect.Placeholder(paramIdx)))
			args = append(args, w.value)
			paramIdx++
		}
	}

	return sb.String(), args
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

	whereSQL, args := q.buildWhere(1)
	if whereSQL != "" {
		sqlStr += " WHERE " + whereSQL
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
	rows, err := q.queryer.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// First returns the first matching row, or nil if none found.
func (q *QueryBuilder) First() (map[string]interface{}, error) {
	clone := q.Clone()
	clone.limitN = 1
	rows, err := clone.Get()
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
	clone := q.Clone()
	clone.columns = []string{"COUNT(*) as cnt"}
	row, err := clone.First()
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return ToInt64(row["cnt"]), nil
}

// PaginationResult encapsulates paginated query results and metadata.
type PaginationResult struct {
	Data        []map[string]interface{} `json:"data"`
	Total       int64                    `json:"total"`
	CurrentPage int                      `json:"current_page"`
	PerPage     int                      `json:"per_page"`
	LastPage    int                      `json:"last_page"`
	From        int                      `json:"from"`
	To          int                      `json:"to"`
}

// Paginate executes a paginated query, computing total rows, last page, and offsets automatically.
func (q *QueryBuilder) Paginate(page, perPage int) (*PaginationResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 15
	}

	total, err := q.Count()
	if err != nil {
		return nil, err
	}

	lastPage := int(math.Ceil(float64(total) / float64(perPage)))
	if lastPage == 0 {
		lastPage = 1
	}

	offset := (page - 1) * perPage
	rows, err := q.Clone().Limit(perPage).Offset(offset).Get()
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []map[string]interface{}{}
	}

	from := 0
	to := 0
	if total > 0 && len(rows) > 0 {
		from = offset + 1
		to = offset + len(rows)
	}

	return &PaginationResult{
		Data:        rows,
		Total:       total,
		CurrentPage: page,
		PerPage:     perPage,
		LastPage:    lastPage,
		From:        from,
		To:          to,
	}, nil
}


// Insert inserts a row from a column->value map and returns the new auto-increment ID.
// For PostgreSQL, it automatically uses "RETURNING id" since LastInsertId is not supported.
func (q *QueryBuilder) Insert(data map[string]interface{}) (int64, error) {
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))
	dialect := q.getDialect()

	idx := 1
	for k, v := range data {
		cols = append(cols, k)
		placeholders = append(placeholders, dialect.Placeholder(idx))
		args = append(args, v)
		idx++
	}

	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q.table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	if !dialect.SupportsLastInsertID() {
		sqlStr += " RETURNING id"
		var newID int64
		err := q.queryer.QueryRow(sqlStr, args...).Scan(&newID)
		return newID, err
	}

	res, err := q.queryer.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Update updates matching rows and returns rows affected.
func (q *QueryBuilder) Update(data map[string]interface{}) (int64, error) {
	sets := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))
	dialect := q.getDialect()

	idx := 1
	for k, v := range data {
		sets = append(sets, fmt.Sprintf("%s = %s", k, dialect.Placeholder(idx)))
		args = append(args, v)
		idx++
	}

	sqlStr := fmt.Sprintf("UPDATE %s SET %s", q.table, strings.Join(sets, ", "))

	whereSQL, whereArgs := q.buildWhere(idx)
	if whereSQL != "" {
		sqlStr += " WHERE " + whereSQL
		args = append(args, whereArgs...)
	}

	res, err := q.queryer.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Delete removes matching rows and returns rows affected.
func (q *QueryBuilder) Delete() (int64, error) {
	sqlStr := fmt.Sprintf("DELETE FROM %s", q.table)
	whereSQL, args := q.buildWhere(1)
	if whereSQL != "" {
		sqlStr += " WHERE " + whereSQL
	}

	res, err := q.queryer.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Pluck returns a slice containing only the values of the requested column.
func (q *QueryBuilder) Pluck(column string) ([]interface{}, error) {
	clone := q.Clone()
	clone.columns = []string{column}
	rows, err := clone.Get()
	if err != nil {
		return nil, err
	}
	out := make([]interface{}, len(rows))
	for i, r := range rows {
		out[i] = r[column]
	}
	return out, nil
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
