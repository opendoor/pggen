// Code generated by pggen. DO NOT EDIT

package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"strings"
	"time"

	"github.com/opendoor-labs/pggen"
)

func genBulkInsertStmt(
	table string,
	fields []string,
	nrecords int,
	pkeyName string,
	includeID bool,
) string {
	var ret strings.Builder

	genInsertCommon(&ret, table, fields, nrecords, pkeyName, includeID)

	ret.WriteString(" RETURNING \"")
	ret.WriteString(pkeyName)
	ret.WriteRune('"')

	return ret.String()
}

// shared between the generated upsert code and genBulkInsertStmt
func genInsertCommon(
	into *strings.Builder,
	table string,
	fields []string,
	nrecords int,
	pkeyName string,
	includeID bool,
) {
	into.WriteString("INSERT INTO \"")
	into.WriteString(table)
	into.WriteString("\" (")
	for i, field := range fields {
		if !includeID && field == pkeyName {
			continue
		}

		into.WriteRune('"')
		into.WriteString(field)
		into.WriteRune('"')
		if i+1 < len(fields) {
			into.WriteRune(',')
		}
	}
	into.WriteString(") VALUES ")

	nInsertFields := len(fields)
	if !includeID {
		nInsertFields--
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
	fields []string,
	fieldMask pggen.FieldSet,
	pkeyName string,
) string {
	var ret strings.Builder

	ret.WriteString("UPDATE \"")
	ret.WriteString(table)
	ret.WriteString("\" SET ")

	lhs := make([]string, 0, len(fields))
	rhs := make([]string, 0, len(fields))
	argNo := 1
	for i, f := range fields {
		if fieldMask.Test(i) {
			lhs = append(lhs, f)
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
	tableName string,
	tab *[]int, // out
) error {
	rows, err := p.topLevelDB.QueryContext(ctx, `
		SELECT a.attname
		FROM pg_attribute a
		JOIN pg_class c ON (c.oid = a.attrelid)
		WHERE a.attisdropped = false AND c.relname = $1 AND a.attnum > 0
	`, tableName)
	if err != nil {
		return err
	}

	type idxMapping struct {
		gen int
		run int
	}
	indicies := []idxMapping{}

	for i := 0; rows.Next(); i++ {
		var colName string
		err = rows.Scan(&colName)
		if err != nil {
			return err
		}

		genIdx, ok := genTimeColIdxTab[colName]
		if !ok {
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

// a type that will accept SQL result and just throw it away
type pggenSinkScanner struct{}

func (s *pggenSinkScanner) Scan(value interface{}) error {
	return nil
}

// PggenPolyNullTime is shipped as sql.NullTime in go 1.13, but
// older versions of go don't have it yet, so we just roll it ourselves
// for compatibility.
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

	t, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("scanning to NullTime: expected time.Time")
	}
	n.Time = t
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

func convertNullUUID(u uuid.NullUUID) *uuid.UUID {
	if u.Valid {
		return &u.UUID
	}
	return nil
}
