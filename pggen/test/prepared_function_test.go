package test

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"regexp"
	"testing"

	_ "github.com/lib/pq"

	"github.com/opendoor-labs/pggen/pggen/test/db_shims"
)

func TestReturnsText(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ReturnsText(ctx)
		},
		expected: "foo",
	}.test(t)
}

func TestConcatText(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ConcatsText(ctx, "foo", "bar")
		},
		expected: `\[.*foobar.*\]`,
	}.test(t)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ConcatsText(ctx, "", "bar")
		},
		expected: `\[.*bar.*\]`,
	}.test(t)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ConcatsText(ctx, "", "")
		},
		expected: `\[.*"".*Valid.*true.*]`,
	}.test(t)
}

func TestSelectStringTypes(t *testing.T) {
	foo := "foo"
	fooPad := "foo                                     "
	expected := []db_shims.SelectStringTypesRow{
		{
			TextField:           sql.NullString{String: foo, Valid: true},
			TextFieldNotNull:    sql.NullString{String: foo, Valid: true},
			VarcharField:        sql.NullString{String: foo, Valid: true},
			VarcharFieldNotNull: sql.NullString{String: foo, Valid: true},
			CharField:           sql.NullString{String: fooPad, Valid: true},
			CharFieldNotNull:    sql.NullString{String: fooPad, Valid: true},
		},
		{
			TextFieldNotNull:    sql.NullString{String: foo, Valid: true},
			VarcharFieldNotNull: sql.NullString{String: foo, Valid: true},
			CharFieldNotNull:    sql.NullString{String: fooPad, Valid: true},
		},
	}
	txt, err := json.Marshal(expected)
	if err != nil {
		t.Error(err)
	}
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectStringTypes(ctx)
		},
		expected: regexp.QuoteMeta(string(txt)),
	}.test(t)
}

func TestSelectMatchingString(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectMatchingString(ctx, "foo")
		},
		expected: `\[.*foo.*\]`,
	}.test(t)
}

func TestSelectMoney(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectMoney(ctx)
		},
		expected: `\[.*3\.50.*3\.50.*\]`,
	}.test(t)
}

func TestSelectTime(t *testing.T) {
	times, err := pgClient.SelectTime(ctx)
	chkErr(t, err)

	ti := times[0]
	if ti.TsField.Time.String() != "1999-01-08 04:05:06 +0000 +0000" {
		t.Fatalf("0: tsfield (actual = '%s')", ti.TsField.Time.String())
	}
	if ti.TsFieldNotNull.String() != "1999-01-08 04:05:06 +0000 +0000" {
		t.Fatalf("0: tsfieldnn (actual = '%s'", ti.TsFieldNotNull.String())
	}

	ti = times[1]
	if ti.TsField.Valid {
		t.Fatalf("1: tsfield unexpectedly valid")
	}
	if ti.TsFieldNotNull.String() != "1999-01-08 04:05:06 +0000 +0000" {
		t.Fatalf("1: tsfieldnn (actual = '%s')", ti.TsFieldNotNull.String())
	}
	// TODO: there is something weird going on with time marshalling. It
	//       works with some postgres versions, but not with all of them.
}

func TestSelectBool(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectBool(ctx)
		},
		expected: `.*true.*false.*Valid":false.*true.*`,
	}.test(t)
}

func TestSelectEnum(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectEnum(ctx)
		},
		expected: `EnumType":"option1.*EnumType":"option2.*Valid":false.*option1`,
	}.test(t)
}

func TestSelectUUID(t *testing.T) {
	id := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectUuid(ctx)
		},
		expected: id + ".*" + id + ".*Valid\":false.*" + id,
	}.test(t)
}

func TestSelectNumbers(t *testing.T) {
	numbers, err := pgClient.SelectNumbers(ctx)
	chkErr(t, err)

	if len(numbers) != 2 {
		t.Fatal("wrong len")
	}

	one := sql.NullInt64{Int64: 1, Valid: true}
	two := sql.NullInt64{Int64: 2, Valid: true}
	three := sql.NullInt64{Int64: 3, Valid: true}
	n154 := sql.NullString{String: "15.4", Valid: true}
	n164 := sql.NullString{String: "16.4", Valid: true}
	n999 := sql.NullString{String: "999", Valid: true}
	n19d99 := sql.NullString{String: "19.99", Valid: true}
	n2d3 := sql.NullFloat64{Float64: 2.3, Valid: true}
	n9d3 := sql.NullFloat64{Float64: 9.3, Valid: true}

	for i, n := range numbers {

		if (i == 0 && !reflect.DeepEqual(n.SmallintField, one)) ||
			!reflect.DeepEqual(n.SmallintFieldNotNull, one) {
			t.Fatalf("%d: small int mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.IntegerField, two)) ||
			!reflect.DeepEqual(n.IntegerFieldNotNull, two) {
			t.Fatalf("%d: integer mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.BigintField, three)) ||
			!reflect.DeepEqual(n.BigintFieldNotNull, three) {
			t.Fatalf("%d: big int mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.DecimalField, n154)) ||
			!reflect.DeepEqual(n.DecimalFieldNotNull, n154) {
			t.Fatalf("%d: decimal mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.NumericField, n164)) ||
			!reflect.DeepEqual(n.NumericFieldNotNull, n164) {
			t.Fatalf("%d: numeric mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.NumericPrecField, n999)) ||
			!reflect.DeepEqual(n.NumericPrecFieldNotNull, n999) {
			t.Fatalf("%d: numeric prec mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.NumericPrecScaleField, n19d99)) ||
			!reflect.DeepEqual(n.NumericPrecScaleFieldNotNull, n19d99) {
			t.Fatalf("%d: numeric prec scale mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.RealField, n2d3)) ||
			!reflect.DeepEqual(n.RealFieldNotNull, n2d3) {
			t.Fatalf("%d: real mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.DoubleField, n9d3)) ||
			!reflect.DeepEqual(n.DoubleFieldNotNull, n9d3) {
			t.Fatalf("%d: real mismatch", i)
		}
	}

	/*
		numberStrings := []string{
			"1", "2", "3", `15\.4`, `16\.4`, "999", `19\.99`,
			`(:?2\.2999|2.3)`, `9\.3`,
		}

		var re strings.Builder
		for _, ns := range numberStrings {
			re.WriteString(ns)
			re.WriteString(".*?")
			re.WriteString(ns)
			re.WriteString(".*?")
		}
		re.WriteString("1.*?1.*?1.*?1.*?},{.*?") // serial fields, divider
		for _, ns := range numberStrings {
			re.WriteString("null.*?")
			re.WriteString(ns)
			re.WriteString(".*?")
		}
		re.WriteString(`2.*?2.*?2.*?2.*?}\]`) // serial fields

		Expectation{
			call: func() (interface{}, error) {
				return pgClient.SelectNumbers(ctx)
			},
			expected: re.String(),
		}.test(t)
	*/
}

func TestSelectBlob(t *testing.T) {
	db := `3q2\+7w==` // base64 encoded 0xdeadbeef
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectBlobs(ctx)
		},
		expected: db + `.*?` + db + `.*?null.*?` + db,
	}.test(t)
}

func TestNamedReturnFunc(t *testing.T) {
	ret1, err := pgClient.GetSmallEntity1(ctx)
	if err != nil {
		t.Fatal(err)
	}

	ret2, err := pgClient.GetSmallEntity2(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(ret1, ret2) {
		t.Fatalf("results not equal (ret1 = %v, ret2 = %v)", ret1, ret2)
	}
}
