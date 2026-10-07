// Package storetest gives integration tests a PostgreSQL database of their
// own: created for the test, migrated with goose, dropped afterwards. Tests
// never touch the demo database, because budget tests insert rows nothing may
// delete (data model section 12, phase 1 server LLD section 5).
package storetest

import "fmt"

// DemoDatabase is the database `make db` creates for the running product.
const DemoDatabase = "outlet_owl"

// CheckNotDemo refuses the demo database by name. New calls it on the
// database a test is about to use, so a broken URL rewrite fails the test
// before any statement runs.
func CheckNotDemo(current string) error {
	if current == DemoDatabase {
		return fmt.Errorf("storetest: refusing to run tests on the demo database %q", current)
	}
	return nil
}
