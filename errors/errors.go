package commonerrors

import "errors"

var (
	ErrCannotOpenDatabase = errors.New("cannot open database")
	ErrDatabaseNotExist   = errors.New("database doesn't exist (need to run `setup`)")
	ErrIDNotInDatabase    = errors.New("user id not in database")
	ErrCannotParse        = errors.New("cannot parse value")
	ErrCannotInsertIntoDB = errors.New("cannot insert value into database")
)
