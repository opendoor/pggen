package test

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/cmd/pggen/test/models"
)

func TestNonNullText(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.NonNullText(ctx)
		},
		expected: `\["foo"\]`,
	}.test(t)
}

func TestMixedNullText(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.MixedNullText(ctx)
		},
		expected: `\["foo",null\]`,
	}.test(t)
}

func TestMultiReturn(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.MultiReturn(ctx)
		},
		expected: `\[.*TextField":"foo".*SmallintField":1.*TextField":null.*SmallintField":null.*\]`,
	}.test(t)
}

func TestTextArg(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.TextArg(ctx, "foo")
		},
		expected: `\["foo"\]`,
	}.test(t)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.TextArg(ctx, "not in the data")
		},
		expected: `\[\]`,
	}.test(t)
}

func TestMoneyArg(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.MoneyArg(ctx, "3.50")
		},
		expected: `\["\$3.50"\]`,
	}.test(t)
}

func TestDateTimeArg(t *testing.T) {
	early := time.Unix(1, 2)
	late := time.Unix(2489738792314, 5)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.DateTimeArg(ctx, early, early, early)
		},
		expected: `\["1999-01-08T04:05:06Z"\]`,
	}.test(t)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.DateTimeArg(ctx, late, late, late)
		},
		expected: `\[\]`,
	}.test(t)
}

func TestBooleanArg(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.BooleanArg(ctx, true)
		},
		expected: `\[true\]`,
	}.test(t)
}

func TestEnumArg(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.EnumArg(ctx, models.EnumTypeOption1)
		},
		expected: `\[1\]`,
	}.test(t)
}

func TestUUIDArg(t *testing.T) {
	id := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.UUIDArg(ctx, uuid.Must(uuid.FromString(id)))
		},
		expected: fmt.Sprintf(`\["%s"\]`, id),
	}.test(t)
}

func TestByteaArg(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ByteaArg(ctx, []byte{0xde, 0xad, 0xbe, 0xef})
		},
		expected: `\["3q2\+7w=="]`, // base64 encoded deadbeef
	}.test(t)
}

func TestNumbersArgs(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.NumberArgs(
				ctx, 0, 0, 0, "0", "0", "0", "0", 0.0, 0.0, 0, 0)
		},
		expected: `\[1\]`,
	}.test(t)
}

func TestNamedReturnQuery(t *testing.T) {
	ret1, err := pgClient.HasNamedReturn1(ctx)
	chkErr(t, err)

	ret2, err := pgClient.HasNamedReturn2(ctx)
	chkErr(t, err)

	if !reflect.DeepEqual(ret1, ret2) {
		t.Fatalf("results not equal (ret1 = %v, ret2 = %v)", ret1, ret2)
	}
}

func TestListText(t *testing.T) {
	ids, err := pgClient.TypeRainbowIDs(ctx)
	chkErr(t, err)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ListText(ctx, ids)
		},
		expected: `\["foo",null\]`,
	}.test(t)
}

func TestRollUpNums(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.RollUpNums(ctx)
		},
		expected: regexp.QuoteMeta(`[{"Ints":[3,null],"Decs":["15.4",null]}]`),
	}.test(t)
}

func TestEnumArrays(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ListEnumAsArray(
				ctx,
				[]models.EnumType{models.EnumTypeOption1, models.EnumTypeOption2},
			)
		},
		expected: regexp.QuoteMeta(`[[2,1]]`),
	}.test(t)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ListEnumAsArrayWithNulls(
				ctx,
				[]models.EnumType{models.EnumTypeOption1, models.EnumTypeOption2},
			)
		},
		expected: regexp.QuoteMeta(`[[1,null]]`),
	}.test(t)
}

func TestQueryErrors(t *testing.T) {
	res, err := pgClient.ForceError(ctx)
	if res != nil {
		t.Fatalf("unexpected result")
	}

	if !strings.Contains(err.Error(), `column "injection" does not exist`) {
		t.Fatalf("unexpected err: %s", err.Error())
	}
}

func TestAllMatchingEnums(t *testing.T) {
	check := func(variants []models.EnumType) {
		matching, err := pgClient.AllMatchingEnums(ctx, variants)
		chkErr(t, err)

		expected := make(map[models.EnumType]bool, len(variants))
		for _, v := range variants {
			expected[v] = true
		}

		if len(matching) != 1 {
			t.Fatalf("should have rolled everything up into one row")
		}
		for _, v := range matching[0] {
			if v == nil || !expected[*v] {
				t.Fatalf("unexpected variant: '%s'", v.String())
			}
		}
	}

	check([]models.EnumType{})
	check([]models.EnumType{models.EnumTypeBlank})
	check([]models.EnumType{models.EnumTypeOption1})
	check([]models.EnumType{models.EnumTypeOption1, models.EnumTypeOption2})
	check([]models.EnumType{models.EnumTypeOption1, models.EnumTypeOption1, models.EnumTypeOption2})
}

func TestJSON(t *testing.T) {
	jsonValues, err := pgClient.SelectJSON(ctx)
	chkErr(t, err)

	for _, v := range jsonValues {
		if !(v.JsonField == nil || bytes.Equal([]byte("5"), *v.JsonField)) {
			t.Fatalf("unexpected json_field value")
		}

		if !(v.JsonbField == nil || bytes.Equal([]byte(`{"bar": "baz"}`), *v.JsonbField)) {
			t.Fatalf("unexpected jsonb_field value")
		}

		if !bytes.Equal([]byte(`[1, 2, "foo", null]`), v.JsonFieldNotNull) {
			t.Fatalf("unexpected json_field_not_null value: %s", string(v.JsonFieldNotNull))
		}

		if !bytes.Equal([]byte(`{"foo": ["bar", 1]}`), v.JsonbFieldNotNull) {
			t.Fatalf("unexpected jsonb_field_not_null value")
		}
	}
}

func TestSingleReturnMultiCol(t *testing.T) {
	row, err := pgClient.SingleResultMultiCol(ctx)
	chkErr(t, err)

	if row.TextFieldNotNull != "foo" {
		t.Fatal("unexpected value")
	}
}

func TestSingleReturnSingleCol(t *testing.T) {
	res, err := pgClient.SingleResultSingleCol(ctx)
	chkErr(t, err)

	if res != "foo" {
		t.Fatal("unexpected value")
	}
}

func TestSingleReturnSingleColNullable(t *testing.T) {
	res, err := pgClient.SingleResultSingleColNullable(ctx)
	chkErr(t, err)

	if *res != "foo" {
		t.Fatal("unexpected value")
	}
}

func TestSingleReturnNotFound(t *testing.T) {
	_, err := pgClient.SingleResultNotFound(ctx)
	if !pggen.IsNotFoundError(err) {
		t.Fatal("found it unexpectedly")
	}
}

func TestNullableArguments(t *testing.T) {
	opt1 := models.EnumTypeOption1

	option1, err := pgClient.SearchForNullableEnum(ctx, &opt1)
	chkErr(t, err)

	if *option1.Value != models.EnumTypeOption1 {
		t.Fatal("exected option1")
	}

	null, err := pgClient.SearchForNullableEnumSingleColResult(ctx, nil)
	chkErr(t, err)
	if null != nil {
		t.Fatal("expected nil")
	}
}

func TestIntervalEcho(t *testing.T) {
	res, err := pgClient.AddHourToInterval(ctx, "1h")
	chkErr(t, err)
	if res != "02:00:00" {
		t.Fatalf("bad result, actual = %s", res)
	}
}
