package test

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/opendoor-labs/pggen/cmd/pggen/test/db_shims"
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
			return pgClient.EnumArg(ctx, db_shims.EnumTypeOption1)
		},
		expected: `\["option1"\]`,
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
				[]db_shims.EnumType{"option1", "option2"},
			)
		},
		expected: regexp.QuoteMeta(`[["option2","option1"]]`),
	}.test(t)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ListEnumAsArrayWithNulls(
				ctx,
				[]db_shims.EnumType{"option1", "option2"},
			)
		},
		expected: regexp.QuoteMeta(`[["option1",null]]`),
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
