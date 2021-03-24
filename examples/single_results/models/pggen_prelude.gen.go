// (c) 2021 Opendoor Labs Inc.
// This code is licenced under the MIT licence (see the LICENCE file in the repo root).
// Code generated by pggen. DO NOT EDIT

package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/jackc/pgconn"
	"strings"
	"sync"
	"time"

	"github.com/opendoor-labs/pggen"
)

type fieldNameAndIdx struct {
	name string
	idx  int
}

func genBulkInsertStmt(
	table string,
	fields []fieldNameAndIdx,
	nrecords int,
	pkeyName string,
	includeID bool,
	defaultFieldSet pggen.FieldSet,
) string {
	var ret strings.Builder

	genInsertCommon(&ret, table, fields, nrecords, pkeyName, includeID, defaultFieldSet)

	ret.WriteString(" RETURNING \"")
	ret.WriteString(pkeyName)
	ret.WriteRune('"')

	return ret.String()
}

func genInsertCommon(
	into *strings.Builder,
	table string,
	fields []fieldNameAndIdx,
	nrecords int,
	pkeyName string,
	includeID bool,
	defaultFieldSet pggen.FieldSet,
) {
	into.WriteString("INSERT INTO ")
	into.WriteString(table)
	into.WriteString(" (")
	for i, field := range fields {
		if (!includeID && field.name == pkeyName) || defaultFieldSet.Test(field.idx) {
			continue
		}

		into.WriteRune('"')
		into.WriteString(field.name)
		into.WriteRune('"')
		if i+1 < len(fields) {
			into.WriteRune(',')
		}
	}
	into.WriteString(") VALUES ")

	nInsertFields := len(fields)
	for _, field := range fields {
		if defaultFieldSet.Test(field.idx) || (!includeID && field.name == pkeyName) {
			nInsertFields--
		}
	}

	nextArg := 1
	for recNo := 0; recNo < nrecords; recNo++ {
		slots := make([]string, 0, nInsertFields)
		for colNo := 1; colNo <= nInsertFields; colNo++ {
			slots = append(
				slots,
				fmt.Sprintf("$%d", nextArg),
			)
			nextArg++
		}
		into.WriteString("(")
		into.WriteString(strings.Join(slots, ", "))
		if recNo < nrecords-1 {
			into.WriteString("),\n")
		} else {
			into.WriteString(")\n")
		}
	}
}

func genUpdateStmt(
	table string,
	pgPkey string,
	fields []fieldNameAndIdx,
	fieldMask pggen.FieldSet,
	pkeyName string,
) string {
	var ret strings.Builder

	ret.WriteString("UPDATE ")
	ret.WriteString(table)
	ret.WriteString(" SET ")

	lhs := make([]string, 0, len(fields))
	rhs := make([]string, 0, len(fields))
	argNo := 1
	for i, f := range fields {
		if fieldMask.Test(i) {
			lhs = append(lhs, f.name)
			rhs = append(rhs, fmt.Sprintf("$%d", argNo))
			argNo++
		}
	}

	if len(lhs) > 1 {
		ret.WriteRune('(')
		for i, f := range lhs {
			ret.WriteRune('"')
			ret.WriteString(f)
			ret.WriteRune('"')
			if i+1 < len(lhs) {
				ret.WriteRune(',')
			}
		}
		ret.WriteRune(')')
	} else {
		ret.WriteRune('"')
		ret.WriteString(lhs[0])
		ret.WriteRune('"')
	}
	ret.WriteString(" = ")
	if len(rhs) > 1 {
		ret.WriteString(parenWrap(strings.Join(rhs, ", ")))
	} else {
		ret.WriteString(rhs[0])
	}
	ret.WriteString(" WHERE \"")
	ret.WriteString(pgPkey)
	ret.WriteString("\" = ")
	ret.WriteString(fmt.Sprintf("$%d", argNo))

	ret.WriteString(" RETURNING \"")
	ret.WriteString(pkeyName)
	ret.WriteRune('"')

	return ret.String()
}

func parenWrap(in string) string {
	return "(" + in + ")"
}

func (p *PGClient) fillColPosTab(
	ctx context.Context,
	genTimeColIdxTab map[string]int,
	rwlock *sync.RWMutex,
	rows *sql.Rows,
	tab *[]int, // out
) error {
	// We need to ensure that writes to the slice header are atomic. We want to
	// aquire the lock sooner rather than later to avoid lots of reader goroutines
	// queuing up computations to compute the position table and causing lock
	// contention.
	rwlock.Lock()
	defer rwlock.Unlock()

	type idxMapping struct {
		gen int
		run int
	}
	indicies := []idxMapping{}

	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("reading column names: %s", err.Error())
	}
	for i, colName := range cols {
		genIdx, inTable := genTimeColIdxTab[colName]
		if !inTable {
			genIdx = -1 // this is a new column
		}

		// shift the indicies to be 0 based
		indicies = append(indicies, idxMapping{gen: genIdx, run: i})
	}

	posTab := make([]int, len(indicies))
	for _, mapping := range indicies {
		posTab[mapping.run] = mapping.gen
	}

	*tab = posTab

	return nil
}

func (p *pgClientImpl) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		if isInvalidCachedPlanError(err) {
			// pgx will have flushed its cache as it bubbled this error up, so
			// let's retry the query once
			return p.db.QueryContext(ctx, query, args...)
		}

		return rows, err
	}
	return rows, err
}

func isInvalidCachedPlanError(err error) bool {
	pgxErr, isPgxErr := err.(*pgconn.PgError)
	if !isPgxErr {
		return false
	}

	return pgxErr.Code == "0A000" &&
		pgxErr.Severity == "ERROR" &&
		pgxErr.Message == "cached plan must not change result type"
}

func convertNullString(s sql.NullString) *string {
	if s.Valid {
		return &s.String
	}
	return nil
}

func convertNullBool(b sql.NullBool) *bool {
	if b.Valid {
		return &b.Bool
	}
	return nil
}

// a type that will accept an SQL result and just throw it away
type pggenSinkScanner struct{}

func (s *pggenSinkScanner) Scan(value interface{}) error {
	return nil
}

// We roll our own time Valuer for two reasons:
//   - sql.NullTime is in go 1.13 which is after our minimum supported
//     go version.
//   - jackc/pgx inexplicably returns a 'string' value for postgres 'time'
//     types rather than a 'time.Time' value as you would expect.
// NOTE: while this Valuer is meant to handle nulls, it can be used
// for non-nullable values as well.
type pggenNullTime struct {
	Time  time.Time
	Valid bool
}

func (n *pggenNullTime) Scan(value interface{}) error {
	if value == nil {
		n.Time, n.Valid = time.Time{}, false
		return nil
	}
	n.Valid = true

	switch t := value.(type) {
	case time.Time:
		n.Time = t
	case string:
		// this is a postgres 'time' type and we are using the jackc/pgx driver
		parsed, err := time.Parse("15:04:05-07", t)
		if err != nil {
			parsed, err = time.Parse("15:04:05", t) // might not have a zone
			if err != nil {
				return fmt.Errorf("parsing pg time: %s", err.Error())
			}
		}
		n.Time = parsed
	default:
		return fmt.Errorf("scanning to NullTime: expected time.Time")
	}
	return nil
}
func (n pggenNullTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

func convertNullTime(t pggenNullTime) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

func convertNullFloat64(f sql.NullFloat64) *float64 {
	if f.Valid {
		return &f.Float64
	}
	return nil
}

func convertNullInt64(i sql.NullInt64) *int64 {
	if i.Valid {
		return &i.Int64
	}
	return nil
}
