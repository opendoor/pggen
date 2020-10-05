package test

import (
	"encoding/json"
	"math"
	"reflect"
	"regexp"
	"testing"

	_ "github.com/lib/pq"

	"github.com/opendoor-labs/pggen/cmd/pggen/test/models"
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
		expected: `\[""\]`,
	}.test(t)
}

func TestSelectStringTypes(t *testing.T) {
	foo := "foo"
	fooPad := "foo                                     "
	expected := []models.SelectStringTypesRow{
		{
			TextField:           &foo,
			TextFieldNotNull:    &foo,
			VarcharField:        &foo,
			VarcharFieldNotNull: &foo,
			CharField:           &fooPad,
			CharFieldNotNull:    &fooPad,
		},
		{
			TextFieldNotNull:    &foo,
			VarcharFieldNotNull: &foo,
			CharFieldNotNull:    &fooPad,
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
	timeStr := ti.TsField.String()
	if !(timeStr == "1999-01-08 04:05:06 +0000 +0000" || timeStr == "1999-01-08 04:05:06 +0000 UTC") {
		t.Fatalf("0: tsfield (actual = '%s')", ti.TsField.String())
	}
	timeStr = ti.TsFieldNotNull.String()
	if !(timeStr == "1999-01-08 04:05:06 +0000 +0000" || timeStr == "1999-01-08 04:05:06 +0000 UTC") {
		t.Fatalf("0: tsfieldnn (actual = '%s'", ti.TsFieldNotNull.String())
	}

	ti = times[1]
	if ti.TsField != nil {
		t.Fatalf("1: tsfield unexpectedly valid")
	}
	timeStr = ti.TsFieldNotNull.String()
	if !(timeStr == "1999-01-08 04:05:06 +0000 +0000" || timeStr == "1999-01-08 04:05:06 +0000 UTC") {
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
		expected: `.*true.*false.*null.*true.*`,
	}.test(t)
}

func TestSelectEnum(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectEnum(ctx)
		},
		expected: `1.*2.*null.*1`,
	}.test(t)
}

func TestSelectUUID(t *testing.T) {
	id := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectUuid(ctx)
		},
		expected: id + ".*" + id + ".*null.*" + id,
	}.test(t)
}

func TestSelectNumbers(t *testing.T) {
	numbers, err := pgClient.SelectNumbers(ctx)
	chkErr(t, err)

	if len(numbers) != 2 {
		t.Fatal("wrong len")
	}

	n154 := "15.4"
	n164 := "16.4"
	n999 := "999"
	n19d99 := "19.99"
	n2d3 := 2.3
	n9d3 := 9.3

	for i, n := range numbers {
		if i == 0 && *n.SmallintField != 1 || *n.SmallintFieldNotNull != 1 {
			t.Fatalf("%d: small int mismatch", i)
		}

		if i == 0 && *n.IntegerField != 2 || *n.IntegerFieldNotNull != 2 {
			t.Fatalf("%d: integer mismatch", i)
		}

		if i == 0 && *n.BigintField != 3 || *n.BigintFieldNotNull != 3 {
			t.Fatalf("%d: big int mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.DecimalField, &n154)) ||
			!reflect.DeepEqual(n.DecimalFieldNotNull, &n154) {
			t.Fatalf("%d: decimal mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.NumericField, &n164)) ||
			!reflect.DeepEqual(n.NumericFieldNotNull, &n164) {
			t.Fatalf("%d: numeric mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.NumericPrecField, &n999)) ||
			!reflect.DeepEqual(n.NumericPrecFieldNotNull, &n999) {
			t.Fatalf("%d: numeric prec mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.NumericPrecScaleField, &n19d99)) ||
			!reflect.DeepEqual(n.NumericPrecScaleFieldNotNull, &n19d99) {
			t.Fatalf("%d: numeric prec scale mismatch", i)
		}

		if (i == 0 && math.Abs(*n.RealField-n2d3) > 0.001) ||
			math.Abs(*n.RealFieldNotNull-n2d3) > 0.001 {
			t.Fatalf("%d: real mismatch", i)
		}

		if (i == 0 && !reflect.DeepEqual(n.DoubleField, &n9d3)) ||
			!reflect.DeepEqual(n.DoubleFieldNotNull, &n9d3) {
			t.Fatalf("%d: double mismatch", i)
		}
	}
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
