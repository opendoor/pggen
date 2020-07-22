package mocks

//go:generate counterfeiter -o=dbconn.mock.go -fake-name=DBConn github.com/opendoor-labs/pggen.DBConn
//go:generate counterfeiter -o=result.mock.go -fake-name=Result database/sql.Result
