package gen

import "testing"

func chkErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
