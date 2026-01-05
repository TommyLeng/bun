package bun

import (
	"context"
	"database/sql"

	"github.com/TommyLeng/bun/schema"
)

type RawQuery struct {
	baseQuery

	query string
	args  []interface{}
	IsSP  bool
	IsSPM bool // SP with multiple result sets
}

// Deprecated: Use NewRaw instead. When add it to IDB, it conflicts with the sql.Conn#Raw
func (db *DB) Raw(query string, args ...interface{}) *RawQuery {
	return &RawQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		query: query,
		args:  args,
	}
}

func NewRawQuery(db *DB, query string, args ...interface{}) *RawQuery {
	return &RawQuery{
		baseQuery: baseQuery{
			db:   db,
			conn: db.DB,
		},
		query: query,
		args:  args,
	}
}

func (q *RawQuery) Conn(db IConn) *RawQuery {
	q.setConn(db)
	return q
}

func (q *RawQuery) Err(err error) *RawQuery {
	q.setErr(err)
	return q
}

func (q *RawQuery) SetSP() *RawQuery {
	q.IsSP = true
	return q
}

// SetSPM sets the query as a stored procedure with multiple result sets.
func (q *RawQuery) SetSPM() *RawQuery {
	q.IsSPM = true
	return q
}

func (q *RawQuery) Exec(ctx context.Context, dest ...interface{}) (sql.Result, error) {
	return q.scanOrExec(ctx, dest, len(dest) > 0)
}

func (q *RawQuery) Scan(ctx context.Context, dest ...interface{}) error {
	_, err := q.scanOrExec(ctx, dest, true)
	return err
}

func (q *RawQuery) scanOrExec(
	ctx context.Context, dest []interface{}, hasDest bool,
) (sql.Result, error) {
	if q.err != nil {
		return nil, q.err
	}

	// Case 1: No destination - just execute (no scan needed)
	if !hasDest {
		if q.IsSP || q.IsSPM {
			return q.execSp(ctx, q, q.query, q.args)
		}
		query := q.db.format(q.query, q.args)
		return q.exec(ctx, q, query)
	}

	// Case 2: Has destination - need to scan results
	if q.IsSPM {
		// SP with multiple result sets
		models, err := q.getModels(dest)
		if err != nil {
			return nil, err
		}
		return q.scanSpMulti(ctx, q, q.query, q.args, models, hasDest)
	}

	if q.IsSP {
		// SP with single result set
		model, err := q.getModel(dest)
		if err != nil {
			return nil, err
		}
		return q.scanSp(ctx, q, q.query, q.args, model, hasDest)
	}

	// Normal query
	model, err := q.getModel(dest)
	if err != nil {
		return nil, err
	}
	query := q.db.format(q.query, q.args)
	return q.scan(ctx, q, query, model, hasDest)
}

func (q *RawQuery) AppendQuery(fmter schema.Formatter, b []byte) ([]byte, error) {
	return fmter.AppendQuery(b, q.query, q.args...), nil
}

func (q *RawQuery) Operation() string {
	return "SELECT"
}
