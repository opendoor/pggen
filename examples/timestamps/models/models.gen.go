// Code generated by pggen DO NOT EDIT

package models

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/opendoor-labs/pggen"
	"github.com/opendoor-labs/pggen/include"
	"github.com/opendoor-labs/pggen/unstable"
	"strings"
	"sync"
	"time"
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
	rwlockForUser                sync.RWMutex
	colIdxTabForUser             []int
	rwlockForGetUserAnywayRow    sync.RWMutex
	colIdxTabForGetUserAnywayRow []int
}

// bogus usage so we can compile with no tables configured
var _ = sync.RWMutex{}

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

func (p *PGClient) GetUser(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*User, error) {
	return p.impl.getUser(ctx, id)
}
func (tx *TxPGClient) GetUser(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*User, error) {
	return tx.impl.getUser(ctx, id)
}
func (p *pgClientImpl) getUser(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*User, error) {
	values, err := p.listUser(ctx, []int64{id}, true /* isGet */)
	if err != nil {
		return nil, err
	}

	// ListUser always returns the same number of records as were
	// requested, so this is safe.
	return &values[0], err
}

func (p *PGClient) ListUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []User, err error) {
	return p.impl.listUser(ctx, ids, false /* isGet */)
}
func (tx *TxPGClient) ListUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []User, err error) {
	return tx.impl.listUser(ctx, ids, false /* isGet */)
}
func (p *pgClientImpl) listUser(
	ctx context.Context,
	ids []int64,
	isGet bool,
	opts ...pggen.ListOpt,
) (ret []User, err error) {
	if len(ids) == 0 {
		return []User{}, nil
	}

	rows, err := p.db.QueryContext(
		ctx,
		"SELECT * FROM \"users\" WHERE \"id\" = ANY($1) AND \"deleted_at\" IS NULL ",
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

	ret = make([]User, 0, len(ids))
	for rows.Next() {
		var value User
		err = value.Scan(ctx, p.client, rows)
		if err != nil {
			return nil, err
		}
		ret = append(ret, value)
	}

	if len(ret) != len(ids) {
		if isGet {
			return nil, &unstable.NotFoundError{
				Msg: "GetUser: record not found",
			}
		} else {
			return nil, &unstable.NotFoundError{
				Msg: fmt.Sprintf(
					"ListUser: asked for %d records, found %d",
					len(ids),
					len(ret),
				),
			}
		}
	}

	return ret, nil
}

// Insert a User into the database. Returns the primary
// key of the inserted row.
func (p *PGClient) InsertUser(
	ctx context.Context,
	value *User,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return p.impl.insertUser(ctx, value, opts...)
}

// Insert a User into the database. Returns the primary
// key of the inserted row.
func (tx *TxPGClient) InsertUser(
	ctx context.Context,
	value *User,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return tx.impl.insertUser(ctx, value, opts...)
}

// Insert a User into the database. Returns the primary
// key of the inserted row.
func (p *pgClientImpl) insertUser(
	ctx context.Context,
	value *User,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	var ids []int64
	ids, err = p.bulkInsertUser(ctx, []User{*value}, opts...)
	if err != nil {
		return
	}

	if len(ids) != 1 {
		err = fmt.Errorf("inserting a User: %d ids (expected 1)", len(ids))
		return
	}

	ret = ids[0]
	return
}

// Insert a list of User. Returns a list of the primary keys of
// the inserted rows.
func (p *PGClient) BulkInsertUser(
	ctx context.Context,
	values []User,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return p.impl.bulkInsertUser(ctx, values, opts...)
}

// Insert a list of User. Returns a list of the primary keys of
// the inserted rows.
func (tx *TxPGClient) BulkInsertUser(
	ctx context.Context,
	values []User,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return tx.impl.bulkInsertUser(ctx, values, opts...)
}

// Insert a list of User. Returns a list of the primary keys of
// the inserted rows.
func (p *pgClientImpl) bulkInsertUser(
	ctx context.Context,
	values []User,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	if len(values) == 0 {
		return []int64{}, nil
	}

	opt := pggen.InsertOptions{}
	for _, o := range opts {
		o(&opt)
	}
	now := time.Now()
	for i := range values {
		createdAt := now.UTC()
		values[i].CreatedAt = createdAt
	}
	for i := range values {
		updatedAt := now.UTC()
		values[i].UpdatedAt = updatedAt
	}

	args := make([]interface{}, 0, 5*len(values))
	for _, v := range values {
		if opt.UsePkey {
			args = append(args, v.Id)
		}
		args = append(args, v.Email)
		args = append(args, v.CreatedAt)
		args = append(args, v.UpdatedAt)
		args = append(args, v.DeletedAt)
	}

	bulkInsertQuery := genBulkInsertStmt(
		"users",
		fieldsForUser,
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
	UserIdFieldIndex        int = 0
	UserEmailFieldIndex     int = 1
	UserCreatedAtFieldIndex int = 2
	UserUpdatedAtFieldIndex int = 3
	UserDeletedAtFieldIndex int = 4
	UserMaxFieldIndex       int = (5 - 1)
)

// A field set saying that all fields in User should be updated.
// For use as a 'fieldMask' parameter
var UserAllFields pggen.FieldSet = pggen.NewFieldSetFilled(5)

var fieldsForUser []string = []string{
	`id`,
	`email`,
	`created_at`,
	`updated_at`,
	`deleted_at`,
}

// Update a User. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (p *PGClient) UpdateUser(
	ctx context.Context,
	value *User,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return p.impl.updateUser(ctx, value, fieldMask)
}

// Update a User. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (tx *TxPGClient) UpdateUser(
	ctx context.Context,
	value *User,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return tx.impl.updateUser(ctx, value, fieldMask)
}
func (p *pgClientImpl) updateUser(
	ctx context.Context,
	value *User,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	if !fieldMask.Test(UserIdFieldIndex) {
		err = fmt.Errorf("primary key required for updates to 'users'")
		return
	}
	now := time.Now().UTC()
	value.UpdatedAt = now
	fieldMask.Set(UserUpdatedAtFieldIndex, true)

	updateStmt := genUpdateStmt(
		"users",
		"id",
		fieldsForUser,
		fieldMask,
		"id",
	)

	args := make([]interface{}, 0, 5)
	if fieldMask.Test(UserIdFieldIndex) {
		args = append(args, value.Id)
	}
	if fieldMask.Test(UserEmailFieldIndex) {
		args = append(args, value.Email)
	}
	if fieldMask.Test(UserCreatedAtFieldIndex) {
		args = append(args, value.CreatedAt)
	}
	if fieldMask.Test(UserUpdatedAtFieldIndex) {
		args = append(args, value.UpdatedAt)
	}
	if fieldMask.Test(UserDeletedAtFieldIndex) {
		args = append(args, value.DeletedAt)
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

// Upsert a User value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (p *PGClient) UpsertUser(
	ctx context.Context,
	value *User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = p.impl.bulkUpsertUser(ctx, []User{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a User value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (tx *TxPGClient) UpsertUser(
	ctx context.Context,
	value *User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = tx.impl.bulkUpsertUser(ctx, []User{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a set of User values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (p *PGClient) BulkUpsertUser(
	ctx context.Context,
	values []User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return p.impl.bulkUpsertUser(ctx, values, constraintNames, fieldMask, opts...)
}

// Upsert a set of User values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (tx *TxPGClient) BulkUpsertUser(
	ctx context.Context,
	values []User,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return tx.impl.bulkUpsertUser(ctx, values, constraintNames, fieldMask, opts...)
}
func (p *pgClientImpl) bulkUpsertUser(
	ctx context.Context,
	values []User,
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

	now := time.Now()
	createdAt := now.UTC()
	for i := range values {
		values[i].CreatedAt = createdAt
	}
	updatedAt := now.UTC()
	for i := range values {
		values[i].UpdatedAt = updatedAt
	}
	fieldMask.Set(UserUpdatedAtFieldIndex, true)

	var stmt strings.Builder
	genInsertCommon(
		&stmt,
		`users`,
		fieldsForUser,
		len(values),
		`id`,
		options.UsePkey,
	)

	setBits := fieldMask.CountSetBits()
	hasConflictAction := setBits > 1 ||
		(setBits == 1 && fieldMask.Test(UserIdFieldIndex) && options.UsePkey) ||
		(setBits == 1 && !fieldMask.Test(UserIdFieldIndex))

	if hasConflictAction {
		stmt.WriteString("ON CONFLICT (")
		stmt.WriteString(strings.Join(constraintNames, ","))
		stmt.WriteString(") DO UPDATE SET ")

		updateCols := make([]string, 0, 5)
		updateExprs := make([]string, 0, 5)
		if options.UsePkey {
			updateCols = append(updateCols, `id`)
			updateExprs = append(updateExprs, `excluded.id`)
		}
		if fieldMask.Test(UserEmailFieldIndex) {
			updateCols = append(updateCols, `email`)
			updateExprs = append(updateExprs, `excluded.email`)
		}
		if fieldMask.Test(UserCreatedAtFieldIndex) {
			updateCols = append(updateCols, `created_at`)
			updateExprs = append(updateExprs, `excluded.created_at`)
		}
		if fieldMask.Test(UserUpdatedAtFieldIndex) {
			updateCols = append(updateCols, `updated_at`)
			updateExprs = append(updateExprs, `excluded.updated_at`)
		}
		if fieldMask.Test(UserDeletedAtFieldIndex) {
			updateCols = append(updateCols, `deleted_at`)
			updateExprs = append(updateExprs, `excluded.deleted_at`)
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

	args := make([]interface{}, 0, 5*len(values))
	for _, v := range values {
		if options.UsePkey {
			args = append(args, v.Id)
		}
		args = append(args, v.Email)
		args = append(args, v.CreatedAt)
		args = append(args, v.UpdatedAt)
		args = append(args, v.DeletedAt)
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

func (p *PGClient) DeleteUser(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDeleteUser(ctx, []int64{id}, opts...)
}
func (tx *TxPGClient) DeleteUser(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDeleteUser(ctx, []int64{id}, opts...)
}

func (p *PGClient) BulkDeleteUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDeleteUser(ctx, ids, opts...)
}
func (tx *TxPGClient) BulkDeleteUser(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDeleteUser(ctx, ids, opts...)
}
func (p *pgClientImpl) bulkDeleteUser(
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
	now := time.Now().UTC()
	var (
		res sql.Result
		err error
	)
	if options.DoHardDelete {
		res, err = p.db.ExecContext(
			ctx,
			"DELETE FROM \"users\" WHERE \"id\" = ANY($1)",
			pq.Array(ids),
		)
	} else {
		res, err = p.db.ExecContext(
			ctx,
			"UPDATE \"users\" SET \"deleted_at\" = $1 WHERE \"id\" = ANY($2)",
			now,
			pq.Array(ids),
		)
	}
	if err != nil {
		return err
	}

	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if nrows != int64(len(ids)) {
		return fmt.Errorf(
			"BulkDeleteUser: %d rows deleted, expected %d",
			nrows,
			len(ids),
		)
	}

	return err
}

var UserAllIncludes *include.Spec = include.Must(include.Parse(
	`users`,
))

func (p *PGClient) UserFillIncludes(
	ctx context.Context,
	rec *User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.privateUserBulkFillIncludes(ctx, []*User{rec}, includes)
}
func (tx *TxPGClient) UserFillIncludes(
	ctx context.Context,
	rec *User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.privateUserBulkFillIncludes(ctx, []*User{rec}, includes)
}

func (p *PGClient) UserBulkFillIncludes(
	ctx context.Context,
	recs []*User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.privateUserBulkFillIncludes(ctx, recs, includes)
}
func (tx *TxPGClient) UserBulkFillIncludes(
	ctx context.Context,
	recs []*User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.privateUserBulkFillIncludes(ctx, recs, includes)
}
func (p *pgClientImpl) privateUserBulkFillIncludes(
	ctx context.Context,
	recs []*User,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	loadedRecordTab := map[string]interface{}{}

	return p.implUserBulkFillIncludes(ctx, recs, includes, loadedRecordTab)
}

func (p *pgClientImpl) implUserBulkFillIncludes(
	ctx context.Context,
	recs []*User,
	includes *include.Spec,
	loadedRecordTab map[string]interface{},
) (err error) {
	if includes.TableName != `users` {
		return fmt.Errorf(
			"expected includes for 'users', got '%s'",
			includes.TableName,
		)
	}

	loadedTab, inMap := loadedRecordTab[`users`]
	if inMap {
		idToRecord := loadedTab.(map[int64]*User)
		for _, r := range recs {
			_, alreadyLoaded := idToRecord[r.Id]
			if !alreadyLoaded {
				idToRecord[r.Id] = r
			}
		}
	} else {
		idToRecord := make(map[int64]*User, len(recs))
		for _, r := range recs {
			idToRecord[r.Id] = r
		}
		loadedRecordTab[`users`] = idToRecord
	}

	return
}

func (p *PGClient) GetUserAnyway(
	ctx context.Context,
	arg1 int64,
) (ret []User, err error) {
	return p.impl.GetUserAnyway(
		ctx,
		arg1,
	)
}
func (tx *TxPGClient) GetUserAnyway(
	ctx context.Context,
	arg1 int64,
) (ret []User, err error) {
	return tx.impl.GetUserAnyway(
		ctx,
		arg1,
	)
}
func (p *pgClientImpl) GetUserAnyway(
	ctx context.Context,
	arg1 int64,
) (ret []User, err error) {
	ret = []User{}

	var rows *sql.Rows
	rows, err = p.GetUserAnywayQuery(
		ctx,
		arg1,
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

	for rows.Next() {
		var row User
		err = row.Scan(ctx, p.client, rows)
		ret = append(ret, row)
	}

	return
}

func (p *PGClient) GetUserAnywayQuery(
	ctx context.Context,
	arg1 int64,
) (*sql.Rows, error) {
	return p.impl.GetUserAnywayQuery(
		ctx,
		arg1,
	)
}
func (tx *TxPGClient) GetUserAnywayQuery(
	ctx context.Context,
	arg1 int64,
) (*sql.Rows, error) {
	return tx.impl.GetUserAnywayQuery(
		ctx,
		arg1,
	)
}
func (p *pgClientImpl) GetUserAnywayQuery(
	ctx context.Context,
	arg1 int64,
) (*sql.Rows, error) {
	return p.db.QueryContext(
		ctx,
		`SELECT * FROM users WHERE id = $1`,
		arg1,
	)
}

type DBQueries interface {
	//
	// automatic CRUD methods
	//

	// User methods
	GetUser(ctx context.Context, id int64, opts ...pggen.GetOpt) (*User, error)
	ListUser(ctx context.Context, ids []int64, opts ...pggen.ListOpt) ([]User, error)
	InsertUser(ctx context.Context, value *User, opts ...pggen.InsertOpt) (int64, error)
	BulkInsertUser(ctx context.Context, values []User, opts ...pggen.InsertOpt) ([]int64, error)
	UpdateUser(ctx context.Context, value *User, fieldMask pggen.FieldSet, opts ...pggen.UpdateOpt) (ret int64, err error)
	UpsertUser(ctx context.Context, value *User, constraintNames []string, fieldMask pggen.FieldSet, opts ...pggen.UpsertOpt) (int64, error)
	BulkUpsertUser(ctx context.Context, values []User, constraintNames []string, fieldMask pggen.FieldSet, opts ...pggen.UpsertOpt) ([]int64, error)
	DeleteUser(ctx context.Context, id int64, opts ...pggen.DeleteOpt) error
	BulkDeleteUser(ctx context.Context, ids []int64, opts ...pggen.DeleteOpt) error
	UserFillIncludes(ctx context.Context, rec *User, includes *include.Spec, opts ...pggen.IncludeOpt) error
	UserBulkFillIncludes(ctx context.Context, recs []*User, includes *include.Spec, opts ...pggen.IncludeOpt) error

	//
	// query methods
	//

	// GetUserAnyway query
	GetUserAnyway(
		ctx context.Context,
		arg1 int64,
	) ([]User, error)
	GetUserAnywayQuery(
		ctx context.Context,
		arg1 int64,
	) (*sql.Rows, error)

	//
	// stored function methods
	//

	//
	// stmt methods
	//

}

type User struct {
	Id        int64      `gorm:"column:id;is_primary"`
	Email     string     `gorm:"column:email"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

func (r *User) Scan(ctx context.Context, client *PGClient, rs *sql.Rows) error {
	client.rwlockForUser.RLock()
	if client.colIdxTabForUser == nil {
		client.rwlockForUser.RUnlock() // release the lock to allow the write lock to be aquired
		err := client.fillColPosTab(
			ctx,
			genTimeColIdxTabForUser,
			&client.rwlockForUser,
			rs,
			&client.colIdxTabForUser,
		)
		if err != nil {
			return err
		}
		client.rwlockForUser.RLock() // get the lock back for the rest of the routine
	}

	var nullableTgts nullableScanTgtsForUser

	scanTgts := make([]interface{}, len(client.colIdxTabForUser))
	for runIdx, genIdx := range client.colIdxTabForUser {
		if genIdx == -1 {
			scanTgts[runIdx] = &pggenSinkScanner{}
		} else {
			scanTgts[runIdx] = scannerTabForUser[genIdx](r, &nullableTgts)
		}
	}
	client.rwlockForUser.RUnlock() // we are now done referencing the idx tab in the happy path

	err := rs.Scan(scanTgts...)
	if err != nil {
		// The database schema may have been changed out from under us, let's
		// check to see if we just need to update our column index tables and retry.
		colNames, colsErr := rs.Columns()
		if colsErr != nil {
			return fmt.Errorf("pggen: checking column names: %s", colsErr.Error())
		}
		client.rwlockForUser.RLock()
		if len(client.colIdxTabForUser) != len(colNames) {
			client.rwlockForUser.RUnlock() // release the lock to allow the write lock to be aquired
			err = client.fillColPosTab(
				ctx,
				genTimeColIdxTabForUser,
				&client.rwlockForUser,
				rs,
				&client.colIdxTabForUser,
			)
			if err != nil {
				return err
			}

			return r.Scan(ctx, client, rs)
		} else {
			client.rwlockForUser.RUnlock()
			return err
		}
	}
	r.CreatedAt = nullableTgts.scanCreatedAt.Time
	r.UpdatedAt = nullableTgts.scanUpdatedAt.Time
	r.DeletedAt = convertNullTime(nullableTgts.scanDeletedAt)

	return nil
}

type nullableScanTgtsForUser struct {
	scanCreatedAt pggenNullTime
	scanUpdatedAt pggenNullTime
	scanDeletedAt pggenNullTime
}

// a table mapping codegen-time col indicies to functions returning a scanner for the
// field that was at that column index at codegen-time.
var scannerTabForUser = [...]func(*User, *nullableScanTgtsForUser) interface{}{
	func(
		r *User,
		nullableTgts *nullableScanTgtsForUser,
	) interface{} {
		return &(r.Id)
	},
	func(
		r *User,
		nullableTgts *nullableScanTgtsForUser,
	) interface{} {
		return &(r.Email)
	},
	func(
		r *User,
		nullableTgts *nullableScanTgtsForUser,
	) interface{} {
		return &(nullableTgts.scanCreatedAt)
	},
	func(
		r *User,
		nullableTgts *nullableScanTgtsForUser,
	) interface{} {
		return &(nullableTgts.scanUpdatedAt)
	},
	func(
		r *User,
		nullableTgts *nullableScanTgtsForUser,
	) interface{} {
		return &(nullableTgts.scanDeletedAt)
	},
}

var genTimeColIdxTabForUser map[string]int = map[string]int{
	`id`:         0,
	`email`:      1,
	`created_at`: 2,
	`updated_at`: 3,
	`deleted_at`: 4,
}
var _ = unstable.NotFoundError{}
