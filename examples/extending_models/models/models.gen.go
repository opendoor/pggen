// Code generated by pggen DO NOT EDIT

package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/lib/pq"
	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/include"
	"github.com/opendoor-labs/pggen/unstable"
	"strings"
)

// PGClient wraps either a 'sql.DB' or a 'sql.Tx'. All pggen-generated
// database access methods for this package are attached to it.
type PGClient struct {
	impl       pgClientImpl
	topLevelDB pggen.DBConn

	// These column indexes are used at run time to enable us to 'SELECT *' against
	// a table that has the same columns in a different order from the ones that we
	// saw in the table we used to generate code. This means that you don't have to worry
	// about migrations merging in a slightly different order than their timestamps have
	// breaking 'SELECT *'.
	colIdxTabForDog []int
}

// NewPGClient creates a new PGClient out of a '*sql.DB' or a
// custom wrapper around a db connection.
//
// If you provide your own wrapper around a '*sql.DB' for logging or
// custom tracing, you MUST forward all calls to an underlying '*sql.DB'
// member of your wrapper.
func NewPGClient(conn pggen.DBConn) *PGClient {
	client := PGClient{
		topLevelDB: conn,
	}
	client.impl = pgClientImpl{
		db:     conn,
		client: &client,
	}

	return &client
}

func (p *PGClient) Handle() pggen.DBHandle {
	return p.topLevelDB
}

func (p *PGClient) BeginTx(ctx context.Context, opts *sql.TxOptions) (*TxPGClient, error) {
	tx, err := p.topLevelDB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &TxPGClient{
		impl: pgClientImpl{
			db:     tx,
			client: p,
		},
	}, nil
}

// A postgres client that operates within a transaction. Supports all the same
// generated methods that PGClient does.
type TxPGClient struct {
	impl pgClientImpl
}

func (tx *TxPGClient) Handle() pggen.DBHandle {
	return tx.impl.db.(*sql.Tx)
}

func (tx *TxPGClient) Rollback() error {
	return tx.impl.db.(*sql.Tx).Rollback()
}

func (tx *TxPGClient) Commit() error {
	return tx.impl.db.(*sql.Tx).Commit()
}

// A database client that can wrap either a direct database connection or a transaction
type pgClientImpl struct {
	db pggen.DBHandle
	// a reference back to the owning PGClient so we can always get at the resolver tables
	client *PGClient
}

func (p *PGClient) GetDog(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*Dog, error) {
	return p.impl.getDog(ctx, id)
}
func (tx *TxPGClient) GetDog(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*Dog, error) {
	return tx.impl.getDog(ctx, id)
}
func (p *pgClientImpl) getDog(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*Dog, error) {
	values, err := p.listDog(ctx, []int64{id}, true /* isGet */)
	if err != nil {
		return nil, err
	}

	// ListDog always returns the same number of records as were
	// requested, so this is safe.
	return &values[0], err
}

func (p *PGClient) ListDog(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []Dog, err error) {
	return p.impl.listDog(ctx, ids, false /* isGet */)
}
func (tx *TxPGClient) ListDog(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []Dog, err error) {
	return tx.impl.listDog(ctx, ids, false /* isGet */)
}
func (p *pgClientImpl) listDog(
	ctx context.Context,
	ids []int64,
	isGet bool,
	opts ...pggen.ListOpt,
) (ret []Dog, err error) {
	if len(ids) == 0 {
		return []Dog{}, nil
	}

	rows, err := p.db.QueryContext(
		ctx,
		"SELECT * FROM \"dogs\" WHERE \"id\" = ANY($1)",
		pq.Array(ids),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			err = rows.Close()
			if err != nil {
				ret = nil
			}
		} else {
			rowErr := rows.Close()
			if rowErr != nil {
				err = fmt.Errorf("%s AND %s", err.Error(), rowErr.Error())
			}
		}
	}()

	ret = make([]Dog, 0, len(ids))
	for rows.Next() {
		var value Dog
		err = value.Scan(ctx, p.client, rows)
		if err != nil {
			return nil, err
		}
		ret = append(ret, value)
	}

	if len(ret) != len(ids) {
		if isGet {
			return nil, &unstable.NotFoundError{
				Msg: "GetDog: record not found",
			}
		} else {
			return nil, &unstable.NotFoundError{
				Msg: fmt.Sprintf(
					"ListDog: asked for %d records, found %d",
					len(ids),
					len(ret),
				),
			}
		}
	}

	return ret, nil
}

// Insert a Dog into the database. Returns the primary
// key of the inserted row.
func (p *PGClient) InsertDog(
	ctx context.Context,
	value *Dog,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return p.impl.insertDog(ctx, value, opts...)
}

// Insert a Dog into the database. Returns the primary
// key of the inserted row.
func (tx *TxPGClient) InsertDog(
	ctx context.Context,
	value *Dog,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return tx.impl.insertDog(ctx, value, opts...)
}

// Insert a Dog into the database. Returns the primary
// key of the inserted row.
func (p *pgClientImpl) insertDog(
	ctx context.Context,
	value *Dog,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	var ids []int64
	ids, err = p.bulkInsertDog(ctx, []Dog{*value}, opts...)
	if err != nil {
		return
	}

	if len(ids) != 1 {
		err = fmt.Errorf("inserting a Dog: %d ids (expected 1)", len(ids))
		return
	}

	ret = ids[0]
	return
}

// Insert a list of Dog. Returns a list of the primary keys of
// the inserted rows.
func (p *PGClient) BulkInsertDog(
	ctx context.Context,
	values []Dog,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return p.impl.bulkInsertDog(ctx, values, opts...)
}

// Insert a list of Dog. Returns a list of the primary keys of
// the inserted rows.
func (tx *TxPGClient) BulkInsertDog(
	ctx context.Context,
	values []Dog,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return tx.impl.bulkInsertDog(ctx, values, opts...)
}

// Insert a list of Dog. Returns a list of the primary keys of
// the inserted rows.
func (p *pgClientImpl) bulkInsertDog(
	ctx context.Context,
	values []Dog,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	if len(values) == 0 {
		return []int64{}, nil
	}

	opt := pggen.InsertOptions{}
	for _, o := range opts {
		o(&opt)
	}

	args := make([]interface{}, 0, 4*len(values))
	for _, v := range values {
		if opt.UsePkey {
			args = append(args, v.Id)
		}
		args = append(args, v.Breed)
		args = append(args, v.Size.String())
		args = append(args, v.AgeInDogYears)
	}

	bulkInsertQuery := genBulkInsertStmt(
		"dogs",
		fieldsForDog,
		len(values),
		"id",
		opt.UsePkey,
	)

	rows, err := p.db.QueryContext(ctx, bulkInsertQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0, len(values))
	for rows.Next() {
		var id int64
		err = rows.Scan(&(id))
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// bit indicies for 'fieldMask' parameters
const (
	DogIdFieldIndex            int = 0
	DogBreedFieldIndex         int = 1
	DogSizeFieldIndex          int = 2
	DogAgeInDogYearsFieldIndex int = 3
	DogMaxFieldIndex           int = (4 - 1)
)

// A field set saying that all fields in Dog should be updated.
// For use as a 'fieldMask' parameter
var DogAllFields pggen.FieldSet = pggen.NewFieldSetFilled(4)

var fieldsForDog []string = []string{
	`id`,
	`breed`,
	`size`,
	`age_in_dog_years`,
}

// Update a Dog. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (p *PGClient) UpdateDog(
	ctx context.Context,
	value *Dog,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return p.impl.updateDog(ctx, value, fieldMask)
}

// Update a Dog. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (tx *TxPGClient) UpdateDog(
	ctx context.Context,
	value *Dog,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return tx.impl.updateDog(ctx, value, fieldMask)
}
func (p *pgClientImpl) updateDog(
	ctx context.Context,
	value *Dog,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	if !fieldMask.Test(DogIdFieldIndex) {
		err = fmt.Errorf("primary key required for updates to 'dogs'")
		return
	}

	updateStmt := genUpdateStmt(
		"dogs",
		"id",
		fieldsForDog,
		fieldMask,
		"id",
	)

	args := make([]interface{}, 0, 4)
	if fieldMask.Test(DogIdFieldIndex) {
		args = append(args, value.Id)
	}
	if fieldMask.Test(DogBreedFieldIndex) {
		args = append(args, value.Breed)
	}
	if fieldMask.Test(DogSizeFieldIndex) {
		args = append(args, value.Size.String())
	}
	if fieldMask.Test(DogAgeInDogYearsFieldIndex) {
		args = append(args, value.AgeInDogYears)
	}

	// add the primary key arg for the WHERE condition
	args = append(args, value.Id)

	var id int64
	err = p.db.QueryRowContext(ctx, updateStmt, args...).
		Scan(&(id))
	if err != nil {
		return
	}

	return id, nil
}

// Upsert a Dog value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (p *PGClient) UpsertDog(
	ctx context.Context,
	value *Dog,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = p.impl.bulkUpsertDog(ctx, []Dog{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a Dog value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (tx *TxPGClient) UpsertDog(
	ctx context.Context,
	value *Dog,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = tx.impl.bulkUpsertDog(ctx, []Dog{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a set of Dog values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (p *PGClient) BulkUpsertDog(
	ctx context.Context,
	values []Dog,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return p.impl.bulkUpsertDog(ctx, values, constraintNames, fieldMask, opts...)
}

// Upsert a set of Dog values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (tx *TxPGClient) BulkUpsertDog(
	ctx context.Context,
	values []Dog,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return tx.impl.bulkUpsertDog(ctx, values, constraintNames, fieldMask, opts...)
}
func (p *pgClientImpl) bulkUpsertDog(
	ctx context.Context,
	values []Dog,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) ([]int64, error) {
	if len(values) == 0 {
		return []int64{}, nil
	}

	options := pggen.UpsertOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if constraintNames == nil || len(constraintNames) == 0 {
		constraintNames = []string{`id`}
	}

	var stmt strings.Builder
	genInsertCommon(
		&stmt,
		`dogs`,
		fieldsForDog,
		len(values),
		`id`,
		options.UsePkey,
	)

	setBits := fieldMask.CountSetBits()
	hasConflictAction := setBits > 1 ||
		(setBits == 1 && fieldMask.Test(DogIdFieldIndex) && options.UsePkey) ||
		(setBits == 1 && !fieldMask.Test(DogIdFieldIndex))

	if hasConflictAction {
		stmt.WriteString("ON CONFLICT (")
		stmt.WriteString(strings.Join(constraintNames, ","))
		stmt.WriteString(") DO UPDATE SET ")

		updateCols := make([]string, 0, 4)
		updateExprs := make([]string, 0, 4)
		if options.UsePkey {
			updateCols = append(updateCols, `id`)
			updateExprs = append(updateExprs, `excluded.id`)
		}
		if fieldMask.Test(DogBreedFieldIndex) {
			updateCols = append(updateCols, `breed`)
			updateExprs = append(updateExprs, `excluded.breed`)
		}
		if fieldMask.Test(DogSizeFieldIndex) {
			updateCols = append(updateCols, `size`)
			updateExprs = append(updateExprs, `excluded.size`)
		}
		if fieldMask.Test(DogAgeInDogYearsFieldIndex) {
			updateCols = append(updateCols, `age_in_dog_years`)
			updateExprs = append(updateExprs, `excluded.age_in_dog_years`)
		}
		if len(updateCols) > 1 {
			stmt.WriteRune('(')
		}
		stmt.WriteString(strings.Join(updateCols, ","))
		if len(updateCols) > 1 {
			stmt.WriteRune(')')
		}
		stmt.WriteString(" = ")
		if len(updateCols) > 1 {
			stmt.WriteRune('(')
		}
		stmt.WriteString(strings.Join(updateExprs, ","))
		if len(updateCols) > 1 {
			stmt.WriteRune(')')
		}
	} else {
		stmt.WriteString("ON CONFLICT DO NOTHING")
	}

	stmt.WriteString(` RETURNING "id"`)

	args := make([]interface{}, 0, 4*len(values))
	for _, v := range values {
		if options.UsePkey {
			args = append(args, v.Id)
		}
		args = append(args, v.Breed)
		args = append(args, v.Size.String())
		args = append(args, v.AgeInDogYears)
	}

	rows, err := p.db.QueryContext(ctx, stmt.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0, len(values))
	for rows.Next() {
		var id int64
		err = rows.Scan(&(id))
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (p *PGClient) DeleteDog(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDeleteDog(ctx, []int64{id}, opts...)
}
func (tx *TxPGClient) DeleteDog(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDeleteDog(ctx, []int64{id}, opts...)
}

func (p *PGClient) BulkDeleteDog(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDeleteDog(ctx, ids, opts...)
}
func (tx *TxPGClient) BulkDeleteDog(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDeleteDog(ctx, ids, opts...)
}
func (p *pgClientImpl) bulkDeleteDog(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	if len(ids) == 0 {
		return nil
	}

	options := pggen.DeleteOptions{}
	for _, o := range opts {
		o(&options)
	}
	res, err := p.db.ExecContext(
		ctx,
		"DELETE FROM \"dogs\" WHERE \"id\" = ANY($1)",
		pq.Array(ids),
	)
	if err != nil {
		return err
	}

	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if nrows != int64(len(ids)) {
		return fmt.Errorf(
			"BulkDeleteDog: %d rows deleted, expected %d",
			nrows,
			len(ids),
		)
	}

	return err
}

var DogAllIncludes *include.Spec = include.Must(include.Parse(
	`dogs`,
))

func (p *PGClient) DogFillIncludes(
	ctx context.Context,
	rec *Dog,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.privateDogBulkFillIncludes(ctx, []*Dog{rec}, includes)
}
func (tx *TxPGClient) DogFillIncludes(
	ctx context.Context,
	rec *Dog,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.privateDogBulkFillIncludes(ctx, []*Dog{rec}, includes)
}

func (p *PGClient) DogBulkFillIncludes(
	ctx context.Context,
	recs []*Dog,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.privateDogBulkFillIncludes(ctx, recs, includes)
}
func (tx *TxPGClient) DogBulkFillIncludes(
	ctx context.Context,
	recs []*Dog,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.privateDogBulkFillIncludes(ctx, recs, includes)
}
func (p *pgClientImpl) privateDogBulkFillIncludes(
	ctx context.Context,
	recs []*Dog,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	loadedRecordTab := map[string]interface{}{}

	return p.implDogBulkFillIncludes(ctx, recs, includes, loadedRecordTab)
}

func (p *pgClientImpl) implDogBulkFillIncludes(
	ctx context.Context,
	recs []*Dog,
	includes *include.Spec,
	loadedRecordTab map[string]interface{},
) (err error) {
	if includes.TableName != `dogs` {
		return fmt.Errorf(
			"expected includes for 'dogs', got '%s'",
			includes.TableName,
		)
	}

	loadedTab, inMap := loadedRecordTab[`dogs`]
	if inMap {
		idToRecord := loadedTab.(map[int64]*Dog)
		for _, r := range recs {
			_, alreadyLoaded := idToRecord[r.Id]
			if !alreadyLoaded {
				idToRecord[r.Id] = r
			}
		}
	} else {
		idToRecord := make(map[int64]*Dog, len(recs))
		for _, r := range recs {
			idToRecord[r.Id] = r
		}
		loadedRecordTab[`dogs`] = idToRecord
	}

	return
}

type DBQueries interface {
	//
	// automatic CRUD methods
	//

	// Dog methods
	GetDog(ctx context.Context, id int64, opts ...pggen.GetOpt) (*Dog, error)
	ListDog(ctx context.Context, ids []int64, opts ...pggen.ListOpt) ([]Dog, error)
	InsertDog(ctx context.Context, value *Dog, opts ...pggen.InsertOpt) (int64, error)
	BulkInsertDog(ctx context.Context, values []Dog, opts ...pggen.InsertOpt) ([]int64, error)
	UpdateDog(ctx context.Context, value *Dog, fieldMask pggen.FieldSet, opts ...pggen.UpdateOpt) (ret int64, err error)
	UpsertDog(ctx context.Context, value *Dog, constraintNames []string, fieldMask pggen.FieldSet, opts ...pggen.UpsertOpt) (int64, error)
	BulkUpsertDog(ctx context.Context, values []Dog, constraintNames []string, fieldMask pggen.FieldSet, opts ...pggen.UpsertOpt) ([]int64, error)
	DeleteDog(ctx context.Context, id int64, opts ...pggen.DeleteOpt) error
	BulkDeleteDog(ctx context.Context, ids []int64, opts ...pggen.DeleteOpt) error
	DogFillIncludes(ctx context.Context, rec *Dog, includes *include.Spec, opts ...pggen.IncludeOpt) error
	DogBulkFillIncludes(ctx context.Context, recs []*Dog, includes *include.Spec, opts ...pggen.IncludeOpt) error

	//
	// query methods
	//

	//
	// stored function methods
	//

	//
	// stmt methods
	//

}

type SizeCategory int

const (
	SizeCategorySmall SizeCategory = iota
	SizeCategoryLarge SizeCategory = iota
)

func (t SizeCategory) String() string {
	switch t {
	case SizeCategorySmall:
		return `small`
	case SizeCategoryLarge:
		return `large`
	default:
		panic(fmt.Sprintf("invalid SizeCategory: %d", t))
	}
}

func SizeCategoryFromString(s string) (SizeCategory, error) {
	var zero SizeCategory

	switch s {
	case `small`:
		return SizeCategorySmall, nil
	case `large`:
		return SizeCategoryLarge, nil
	default:
		return zero, fmt.Errorf("SizeCategory unknown variant '%s'", s)
	}
}

type ScanIntoSizeCategory struct {
	value *SizeCategory
}

func (s *ScanIntoSizeCategory) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("unexpected NULL SizeCategory")
	}

	buff, ok := value.([]byte)
	if !ok {
		return fmt.Errorf(
			"ScanIntoSizeCategory.Scan: expected a []byte",
		)
	}

	val, err := SizeCategoryFromString(string(buff))
	if err != nil {
		return fmt.Errorf("NullSizeCategory.Scan: %s", err.Error())
	}

	*s.value = val

	return nil
}

type NullSizeCategory struct {
	SizeCategory SizeCategory
	Valid        bool
}

// Scan implements the sql.Scanner interface
func (n *NullSizeCategory) Scan(value interface{}) error {
	if value == nil {
		n.SizeCategory, n.Valid = SizeCategory(0), false
		return nil
	}
	buff, ok := value.([]byte)
	if !ok {
		return fmt.Errorf(
			"NullSizeCategory.Scan: expected a []byte",
		)
	}

	val, err := SizeCategoryFromString(string(buff))
	if err != nil {
		return fmt.Errorf("NullSizeCategory.Scan: %s", err.Error())
	}

	n.Valid = true
	n.SizeCategory = val
	return nil
}

// Value implements the sql.Valuer interface
func (n NullSizeCategory) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.SizeCategory.String(), nil
}
func convertNullSizeCategory(v NullSizeCategory) *SizeCategory {
	if v.Valid {
		ret := SizeCategory(v.SizeCategory)
		return &ret
	}
	return nil
}

type Dog struct {
	Id            int64        `gorm:"column:id;is_primary"`
	Breed         string       `gorm:"column:breed"`
	Size          SizeCategory `gorm:"column:size"`
	AgeInDogYears int64        `gorm:"column:age_in_dog_years"`
}

func (r *Dog) Scan(ctx context.Context, client *PGClient, rs *sql.Rows) error {
	if client.colIdxTabForDog == nil {
		err := client.fillColPosTab(
			ctx,
			genTimeColIdxTabForDog,
			`dogs`,
			&client.colIdxTabForDog,
		)
		if err != nil {
			return err
		}
	}

	var nullableTgts nullableScanTgtsForDog

	scanTgts := make([]interface{}, len(client.colIdxTabForDog))
	for runIdx, genIdx := range client.colIdxTabForDog {
		if genIdx == -1 {
			scanTgts[runIdx] = &pggenSinkScanner{}
		} else {
			scanTgts[runIdx] = scannerTabForDog[genIdx](r, &nullableTgts)
		}
	}

	err := rs.Scan(scanTgts...)
	if err != nil {
		// The database schema may have been changed out from under us, let's
		// check to see if we just need to update our column index tables and retry.
		colNames, colsErr := rs.Columns()
		if colsErr != nil {
			return fmt.Errorf("pggen: checking column names: %s", colsErr.Error())
		}
		if len(client.colIdxTabForDog) != len(colNames) {
			err = client.fillColPosTab(
				ctx,
				genTimeColIdxTabForDog,
				`drop_cols`,
				&client.colIdxTabForDog,
			)
			if err != nil {
				return err
			}

			return r.Scan(ctx, client, rs)
		} else {
			return err
		}
	}

	return nil
}

type nullableScanTgtsForDog struct {
}

// a table mapping codegen-time col indicies to functions returning a scanner for the
// field that was at that column index at codegen-time.
var scannerTabForDog = [...]func(*Dog, *nullableScanTgtsForDog) interface{}{
	func(
		r *Dog,
		nullableTgts *nullableScanTgtsForDog,
	) interface{} {
		return &(r.Id)
	},
	func(
		r *Dog,
		nullableTgts *nullableScanTgtsForDog,
	) interface{} {
		return &(r.Breed)
	},
	func(
		r *Dog,
		nullableTgts *nullableScanTgtsForDog,
	) interface{} {
		return &ScanIntoSizeCategory{value: &r.Size}
	},
	func(
		r *Dog,
		nullableTgts *nullableScanTgtsForDog,
	) interface{} {
		return &(r.AgeInDogYears)
	},
}

var genTimeColIdxTabForDog map[string]int = map[string]int{
	`id`:               0,
	`breed`:            1,
	`size`:             2,
	`age_in_dog_years`: 3,
}
