package test

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
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
	// TODO: there is something weird going on with time marshalling. It
	//       works with some postgres versions, but not with all of them.
	expected := regexp.QuoteMeta(`[{"TsField":"1999-01-08T04:05:06Z","TsFieldNotNull":"1999-01-08T04:05:06Z","TszField":"`) +
		`.+` + regexp.QuoteMeta(`","TszFieldNotNull":"`) +
		`.+` + regexp.QuoteMeta(`","DateField":"1995-05-19T00:00:00Z","DateFieldNotNull":"1995-05-19T00:00:00Z","TimeField":"0000-01-01T03:11:21Z","TimeFieldNotNull":"0000-01-01T03:11:21Z","TimezField":"`) + `.+` +
		regexp.QuoteMeta(`","TimezFieldNotNull":"`) + `.+` +
		regexp.QuoteMeta(`"},{"TsField":null,"TsFieldNotNull":"1999-01-08T04:05:06Z","TszField":null,"TszFieldNotNull":"`) + `.+` +
		regexp.QuoteMeta(`","DateField":null,"DateFieldNotNull":"1995-05-19T00:00:00Z","TimeField":null,"TimeFieldNotNull":"0000-01-01T03:11:21Z","TimezField":null,"TimezFieldNotNull":"`) +
		`.+` + regexp.QuoteMeta(`"}]`)

	Expectation{
		call: func() (interface{}, error) {
			return pgClient.SelectTime(ctx)
		},
		expected: expected,
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
