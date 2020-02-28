package test

import (
	"fmt"
	"reflect"
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
		expected: `\[.*String":"foo".*Valid":true.*Valid":false.*\]`,
	}.test(t)
}

func TestMultiReturn(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.MultiReturn(ctx)
		},
		expected: `\[.*String":"foo".*Int64":1.*Valid":false.*Valid":false.*\]`,
	}.test(t)
}

func TestTextArg(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.TextArg(ctx, "foo")
		},
		expected: `\[.*"String":"foo".*\]`,
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
		expected: `\[.*String":"\$3.50".*\]`,
	}.test(t)
}

func TestDateTimeArg(t *testing.T) {
	early := time.Unix(1, 2)
	late := time.Unix(2489738792314, 5)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.DateTimeArg(ctx, early, early, early)
		},
		expected: `\[.*"Time":"1999-01-08T04:05:06Z".*\]`,
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
		expected: `\[.*"Bool":true.*\]`,
	}.test(t)
}

func TestEnumArg(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.EnumArg(ctx, db_shims.EnumTypeOption1)
		},
		expected: `\[.*EnumType":"option1","Valid":true.*\]`,
	}.test(t)
}

func TestUUIDArg(t *testing.T) {
	id := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.UUIDArg(ctx, uuid.Must(uuid.FromString(id)))
		},
		expected: fmt.Sprintf(`\[.*"UUID":"%s".*\]`, id),
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
		expected: `\[.*Int64":1.*\]`,
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
		expected: `\[.*String":"foo".*Valid":true.*Valid":false.*\]`,
	}.test(t)
}

func TestRollUpNums(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.RollUpNums(ctx)
		},
		expected: `Int64":3.*Int64":0.*String":"15.4.*String":"".*`,
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
		expected: `"option2","Valid":true.*"option1","Valid":true`,
	}.test(t)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.ListEnumAsArrayWithNulls(
				ctx,
				[]db_shims.EnumType{"option1", "option2"},
			)
		},
		expected: `"option1","Valid":true.*"","Valid":false`,
	}.test(t)
}
