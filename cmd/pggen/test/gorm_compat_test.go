package test

import (
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"

	"github.com/opendoor-labs/pggen/cmd/pggen/test/models"
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
	var smallEntity models.SmallEntity

	chkErr(t, gormDB.Model(&models.SmallEntity{}).
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
	var smallEntity models.SmallEntity

	chkErr(t, gormDB.Model(&models.SmallEntity{}).
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
		if a.Value == nil {
			t.Fatalf("unexpected null")
		}

		if !allowedValues[*a.Value] {
			t.Fatalf("unexpected value: '%s'", *a.Value)
		}
	}
}

// This one is not strictly gorm related, but in practice it will probably mostly
// be used for gorm compatibility.
func TestCustomAnnotations(t *testing.T) {
	field, ok := reflect.TypeOf(&models.CustomDefaultUuid{}).Elem().FieldByName("Uuid")
	if !ok {
		t.Fatal("field no found")
	}

	if !strings.Contains(field.Tag.Get("customtag"), "my-custom-tag") {
		t.Fatal("missing tag")
	}
}

func Test1ToManyForeignKey(t *testing.T) {
	// load some data into the db
	wackyRootID, err := pgClient.InsertWackyRoot(ctx, &models.WackyRoot{
		Value: "root",
	})
	chkErr(t, err)
	defer pgClient.DeleteWackyRoot(ctx, wackyRootID) // nolint: errcheck
	attachment1ID, err := pgClient.InsertWackyAttachment(ctx, &models.WackyAttachment{
		Value:    "foo",
		WackyRef: wackyRootID,
	})
	chkErr(t, err)
	defer pgClient.DeleteWackyAttachment(ctx, attachment1ID) // nolint: errcheck
	attachment2ID, err := pgClient.InsertWackyAttachment(ctx, &models.WackyAttachment{
		Value:    "bar",
		WackyRef: wackyRootID,
	})
	chkErr(t, err)
	defer pgClient.DeleteWackyAttachment(ctx, attachment2ID) // nolint: errcheck

	// preload the attachments
	var roots []models.WackyRoot
	err = gormDB.Preload("WackyAttachments").Find(&roots).Where("id = ?", wackyRootID).Error
	chkErr(t, err)

	if len(roots[0].WackyAttachments) != 2 {
		log.Fatal("wrong number of attachments loaded")
	}
}

func Test1To1ForeignKey(t *testing.T) {
	wackyRootID, err := pgClient.InsertWackyRoot(ctx, &models.WackyRoot{
		Value: "root",
	})
	chkErr(t, err)
	defer pgClient.DeleteWackyRoot(ctx, wackyRootID) // nolint: errcheck
	attachmentID, err := pgClient.InsertWackySingleAttachment(ctx, &models.WackySingleAttachment{
		Value:    "foo",
		WackyRef: wackyRootID,
	})
	chkErr(t, err)
	defer pgClient.DeleteWackySingleAttachment(ctx, attachmentID) // nolint: errcheck

	var roots []models.WackyRoot
	err = gormDB.Preload("WackySingleAttachment").Find(&roots).Where("id = ?", wackyRootID).Error
	chkErr(t, err)

	if roots[0].WackySingleAttachment == nil {
		log.Fatal("failed to load single attachemnt")
	}
}

func TestGormDefault(t *testing.T) {
	gormDefault := models.GormDefault{}
	gormDB.Create(&gormDefault)

	var zero uuid.UUID
	if zero == gormDefault.Uuid {
		t.Fatal("uuid was not filled in")
	}
}
