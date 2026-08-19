package core

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"
)

// Model is implemented by every entity struct so the generic Repository
// knows which table to query. Example:
//
//	type Product struct {
//	    ID    int64  `db:"id"`
//	    Name  string `db:"name"`
//	    Price float64 `db:"price"`
//	}
//	func (Product) TableName() string { return "products" }
type Model interface {
	TableName() string
}

// Repository is a generic Active-Record-ish accessor: Repository[Product]
// gives you Find/All/Where/Create/Update/Delete without hand-writing SQL
// for each entity, similar to NovaFlow PHP's `ProductModel::all()`.
type Repository[T Model] struct {
	db *DB
}

func NewRepository[T Model](db *DB) *Repository[T] {
	return &Repository[T]{db: db}
}

func (r *Repository[T]) tableName() string {
	var zero T
	return zero.TableName()
}

// Find returns the row with the given primary key, or (nil, nil) if not found.
func (r *Repository[T]) Find(id interface{}) (*T, error) {
	row, err := r.db.Table(r.tableName()).Where("id", "=", id).First()
	if err != nil || row == nil {
		return nil, err
	}
	var entity T
	if err := mapToStruct(row, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// All returns every row in the table.
func (r *Repository[T]) All() ([]T, error) {
	rows, err := r.db.Table(r.tableName()).Get()
	if err != nil {
		return nil, err
	}
	return rowsToStructs[T](rows)
}

// Where starts a filtered query that still returns typed structs via .Get().
type TypedQuery[T Model] struct {
	qb *QueryBuilder
}

func (r *Repository[T]) Where(column, op string, value interface{}) *TypedQuery[T] {
	return &TypedQuery[T]{qb: r.db.Table(r.tableName()).Where(column, op, value)}
}

func (tq *TypedQuery[T]) Where(column, op string, value interface{}) *TypedQuery[T] {
	tq.qb.Where(column, op, value)
	return tq
}

func (tq *TypedQuery[T]) OrderBy(column, dir string) *TypedQuery[T] {
	tq.qb.OrderBy(column, dir)
	return tq
}

func (tq *TypedQuery[T]) Limit(n int) *TypedQuery[T] {
	tq.qb.Limit(n)
	return tq
}

func (tq *TypedQuery[T]) Get() ([]T, error) {
	rows, err := tq.qb.Get()
	if err != nil {
		return nil, err
	}
	return rowsToStructs[T](rows)
}

func (tq *TypedQuery[T]) First() (*T, error) {
	row, err := tq.qb.First()
	if err != nil || row == nil {
		return nil, err
	}
	var entity T
	if err := mapToStruct(row, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// TypedPaginationResult encapsulates paginated query results with typed entity structs.
type TypedPaginationResult[T Model] struct {
	Data        []T   `json:"data"`
	Total       int64 `json:"total"`
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	LastPage    int   `json:"last_page"`
	From        int   `json:"from"`
	To          int   `json:"to"`
}

// Paginate executes a paginated query on the typed query, mapping rows to entity structs.
func (tq *TypedQuery[T]) Paginate(page, perPage int) (*TypedPaginationResult[T], error) {
	pgRes, err := tq.qb.Paginate(page, perPage)
	if err != nil {
		return nil, err
	}

	items, err := rowsToStructs[T](pgRes.Data)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []T{}
	}

	return &TypedPaginationResult[T]{
		Data:        items,
		Total:       pgRes.Total,
		CurrentPage: pgRes.CurrentPage,
		PerPage:     pgRes.PerPage,
		LastPage:    pgRes.LastPage,
		From:        pgRes.From,
		To:          pgRes.To,
	}, nil
}

// Paginate returns paginated typed records directly from the repository.
func (r *Repository[T]) Paginate(page, perPage int) (*TypedPaginationResult[T], error) {
	return (&TypedQuery[T]{qb: r.db.Table(r.tableName())}).Paginate(page, perPage)
}

// Create inserts entity (skipping a zero-value "id" field so MySQL can
// auto-increment it) and returns the new ID.
func (r *Repository[T]) Create(entity *T) (int64, error) {

	data := structToMap(entity, true)
	return r.db.Table(r.tableName()).Insert(data)
}

// Update saves all fields of entity back to its row, matched by "id".
func (r *Repository[T]) Update(entity *T) (int64, error) {
	data := structToMap(entity, false)
	id, ok := data["id"]
	if !ok {
		return 0, errors.New("model: Update requires an id field")
	}
	delete(data, "id")
	return r.db.Table(r.tableName()).Where("id", "=", id).Update(data)
}

// Delete removes the row with the given id.
func (r *Repository[T]) Delete(id interface{}) (int64, error) {
	return r.db.Table(r.tableName()).Where("id", "=", id).Delete()
}

// --- reflection helpers & metadata caching ------------------------------

type structFieldInfo struct {
	index int
	name  string
	dbCol string
}

type structMeta struct {
	fields []structFieldInfo
}

var structMetaCache sync.Map

func getStructMeta(t reflect.Type) *structMeta {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if val, ok := structMetaCache.Load(t); ok {
		return val.(*structMeta)
	}

	var fields []structFieldInfo
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		col := field.Tag.Get("db")
		if col == "" || col == "-" {
			continue
		}
		fields = append(fields, structFieldInfo{
			index: i,
			name:  field.Name,
			dbCol: col,
		})
	}
	meta := &structMeta{fields: fields}
	structMetaCache.Store(t, meta)
	return meta
}

func rowsToStructs[T Model](rows []map[string]interface{}) ([]T, error) {
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		var entity T
		if err := mapToStruct(row, &entity); err != nil {
			return nil, err
		}
		out = append(out, entity)
	}
	return out, nil
}

// mapToStruct copies a DB row (column -> value) into a struct using cached `db:"col"`
// metadata, converting common SQL driver types into the destination field's Go type.
func mapToStruct(row map[string]interface{}, dest interface{}) error {
	v := reflect.ValueOf(dest).Elem()
	meta := getStructMeta(v.Type())

	for _, fInfo := range meta.fields {
		raw, ok := row[fInfo.dbCol]
		if !ok || raw == nil {
			continue
		}
		fv := v.Field(fInfo.index)
		if err := setField(fv, raw); err != nil {
			return fmt.Errorf("field %s: %w", fInfo.name, err)
		}
	}
	return nil
}

func setField(fv reflect.Value, raw interface{}) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(toString(raw))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fv.SetInt(ToInt64(raw))
	case reflect.Float32, reflect.Float64:
		switch n := raw.(type) {
		case float64:
			fv.SetFloat(n)
		case string:
			f, _ := strconv.ParseFloat(n, 64)
			fv.SetFloat(f)
		default:
			fv.SetFloat(0)
		}
	case reflect.Bool:
		switch n := raw.(type) {
		case bool:
			fv.SetBool(n)
		case int64:
			fv.SetBool(n != 0)
		case string:
			b, _ := strconv.ParseBool(n)
			fv.SetBool(b)
		}
	case reflect.Struct:
		if fv.Type() == reflect.TypeOf(time.Time{}) {
			switch n := raw.(type) {
			case time.Time:
				fv.Set(reflect.ValueOf(n))
			case string:
				if ts, err := time.Parse("2006-01-02 15:04:05", n); err == nil {
					fv.Set(reflect.ValueOf(ts))
				} else if ts, err := time.Parse(time.RFC3339, n); err == nil {
					fv.Set(reflect.ValueOf(ts))
				} else if ts, err := time.Parse("2006-01-02", n); err == nil {
					fv.Set(reflect.ValueOf(ts))
				}
			}
		}
	}

	return nil
}

func toString(raw interface{}) string {
	switch n := raw.(type) {
	case string:
		return n
	case []byte:
		return string(n)
	default:
		return fmt.Sprintf("%v", n)
	}
}

// structToMap converts a struct into a column->value map using cached `db:"col"`
// metadata. When skipEmptyID is true, an "id" field left at its zero value is
// omitted so INSERT lets the database auto-increment it.
func structToMap(entity interface{}, skipEmptyID bool) map[string]interface{} {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	meta := getStructMeta(v.Type())
	out := map[string]interface{}{}

	for _, fInfo := range meta.fields {
		fv := v.Field(fInfo.index)
		if fInfo.dbCol == "id" && skipEmptyID && fv.IsZero() {
			continue
		}
		out[fInfo.dbCol] = fv.Interface()
	}
	return out
}

