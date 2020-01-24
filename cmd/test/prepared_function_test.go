package test

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/opendoor-labs/pggen/test/db_shims"
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
		expected: `\[""]`,
	}.test(t)
}

func TestSelectStringTypes(t *testing.T) {
	foo := "foo"
	fooPad := "foo                                     "
	expected := []db_shims.SelectStringTypesRow{
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
	mustTime := func(format string, timeString string) *time.Time {
		ti, err := time.Parse(format, timeString)
		if err != nil {
			t.Error(err)
		}
		return &ti
	}
	expected := []db_shims.SelectTimeRow{
		{
			TsField: mustTime("2006-01-02 15:04:05", "1999-01-08 04:05:06"),
			TsFieldNotNull: *mustTime(
				"2006-01-02 15:04:05",
				"1999-01-08 04:05:06",
			),
			TszField: mustTime(
				"2006-01-02 15:04:05 -0700 MST",
				"1999-01-07 20:05:06 -0500 EST",
			),
			TszFieldNotNull: *mustTime(
				"2006-01-02 15:04:05 -0700 MST",
				"1999-01-07 20:05:06 -0500 EST",
			),
			DateField:         mustTime("2006-01-02", "1995-05-19"),
			DateFieldNotNull:  *mustTime("2006-01-02", "1995-05-19"),
			TimeField:         mustTime("15:04:05", "03:11:21"),
			TimeFieldNotNull:  *mustTime("15:04:05", "03:11:21"),
			TimezField:        mustTime("15:04:05 -0700", "08:00:00 +0300"),
			TimezFieldNotNull: *mustTime("15:04:05 -0700", "08:00:00 +0300"),
		},
		{
			TsFieldNotNull: *mustTime(
				"2006-01-02 15:04:05",
				"1999-01-08 04:05:06",
			),
			TszFieldNotNull: *mustTime(
				"2006-01-02 15:04:05 -0700 MST",
				"1999-01-07 20:05:06 -0500 EST",
			),
			DateFieldNotNull:  *mustTime("2006-01-02", "1995-05-19"),
			TimeFieldNotNull:  *mustTime("15:04:05", "03:11:21"),
			TimezFieldNotNull: *mustTime("15:04:05 -0700", "08:00:00 +0300"),
		},
	}
	txt, err := json.Marshal(expected)
	if err != nil {
		t.Error(err)
	}

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectTime(ctx)
		},
		expected: regexp.QuoteMeta(string(txt)),
	}.test(t)
}

func TestSelectBool(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectBool(ctx)
		},
		expected: `\[\{.*true.*false.*\}.*\{.*null.*true.*\}\]`,
	}.test(t)
}

func TestSelectEnum(t *testing.T) {
	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectEnum(ctx)
		},
		expected: `option1.*option2"},{".*null.*option1`,
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
	numberStrings := []string{
		"1", "2", "3", `15\.4`, `16\.4`, "999", `19\.99`, `2\.2999`, `9\.3`,
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
