package test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/opendoor-labs/pggen/cmd/pggen/test/models"
)

// file: race_test.go
//
// this file contains tests that attempt to exercise potentially racy code
//

// NOTE: I have been able to confirm that this test will reliably turn up
//       data races by commenting out all the lock calls in the generated Scan routine
//       for OffsetTableFilling and then running `go test -race --run TestOffsetTableFilling`
//       in a loop. Re-generating the models code makes the race warnings go away.
func TestOffsetTableFilling(t *testing.T) {
	nscanners := 50
	nmods := 10

	// insert some data so the results are not empty, not really needed but somehow
	// makes me fell better.
	id, err := pgClient.InsertOffsetTableFilling(ctx, &models.OffsetTableFilling{
		I1: 1,
	})
	chkErr(t, err)

	// start mucking about with the table
	errchan := make(chan error)
	modRoutine := func() {
		for i := 0; i < nmods; i++ {
			_, err := pgClient.Handle().ExecContext(
				ctx, fmt.Sprintf("ALTER TABLE offset_table_fillings ADD COLUMN i%d integer", i+2))
			errchan <- err
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(nscanners)
	scanRoutine := func(tid int) {
		for i := 0; i < nscanners; i++ {
			// don't check the error, as it might be something like
			// "sorry, too many clients already" or "cached plan must not change result type".
			// we don't actually care about these issues, we just want to see if we will
			// get a race with `go test -race`
			pgClient.GetOffsetTableFilling(ctx, id) // nolint: errcheck
		}
		wg.Done()
	}

	for i := 0; i < (nscanners / 2); i++ {
		go scanRoutine(i)
	}
	go modRoutine()
	for i := 0; i < (nscanners / 2); i++ {
		go scanRoutine((nscanners / 2) + i)
	}

	wg.Wait()
	for i := 0; i < nmods; i++ {
		err := <-errchan
		chkErr(t, err)
	}

	_, err = pgClient.Handle().ExecContext(ctx, `
	DROP TABLE offset_table_fillings;
	CREATE TABLE offset_table_fillings (
		id SERIAL PRIMARY KEY,
		i1 integer NOT NULL
	);
	`)
	chkErr(t, err)
}
