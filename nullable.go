package main

import (
	"database/sql"
	"time"
)

type sqlNullInt64 = sql.NullInt64
type sqlNullString = sql.NullString
type sqlNullTime = sql.NullTime

func nullInt64(v int64) sqlNullInt64 {
	return sqlNullInt64{Int64: v, Valid: v > 0}
}

func nullTime(t time.Time) sqlNullTime {
	return sqlNullTime{Time: t, Valid: !t.IsZero()}
}
