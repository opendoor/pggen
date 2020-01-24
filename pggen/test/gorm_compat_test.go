package test

import (
	"log"
	"testing"

	"github.com/jinzhu/gorm"

	"github.com/opendoor-labs/pggen/pggen/test/db_shims"
)

var gormDB *gorm.DB

func init() {
	var err error
	gormDB, err = gorm.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
}

// A basic smoke test making sure we can fetch a pggen generated record
// with GORM.
func TestGormGetSmallEntity(t *testing.T) {
	var smallEntity db_shims.SmallEntity

	chkErr(t, gormDB.Model(&db_shims.SmallEntity{}).
		First(&smallEntity).Error)
	if smallEntity.Anint != 17 {
		t.Fatalf("anint = %d, expected 17", smallEntity.Anint)
	}

	if len(smallEntity.Attachments) != 0 {
		t.Fatalf("unexpected attachments")
	}
}

// Make sure that using the Preload routine to load child objects works.
func TestGormPreload(t *testing.T) {
	var smallEntity db_shims.SmallEntity

	chkErr(t, gormDB.Model(&db_shims.SmallEntity{}).
		Preload("Attachments").
		First(&smallEntity).Error)

	if len(smallEntity.Attachments) != 3 {
		t.Fatalf(
			"len(smallEntity.Attachments) = %d, not 3",
			len(smallEntity.Attachments),
		)
	}
	allowedValues := map[string]bool{
		"text 1": true,
		"text 2": true,
		"text 3": true,
	}
	for _, a := range smallEntity.Attachments {
		if !allowedValues[*a.Value] {
			t.Fatalf("unexpected value: '%s'", *a.Value)
		}
	}
}
