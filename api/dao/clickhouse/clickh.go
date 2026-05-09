package clickhouse

import (
	"database/sql"
	"reflect"
)

const InitialCapacity = 256

type ScanRowFunc func(row *sql.Rows) (interface{}, error)

func ScanAllAsync(psql string, srf ScanRowFunc, results chan interface{}, args ...interface{}) (err error) {
	return CK.ScanAllAsync(psql, srf, results, args...)
}

func ScanAll(psql string, srf ScanRowFunc, args ...interface{}) (ret interface{}, err error) {
	return CK.ScanAll(psql, srf, args...)
}

func ScanOne(psql string, srf ScanRowFunc, args ...interface{}) (ret interface{}, err error) {
	return CK.ScanOne(psql, srf, args...)
}

type clickhImpl struct {
	*sql.DB
}

func (m *clickhImpl) ScanAll(psql string, srf ScanRowFunc, args ...interface{}) (ret interface{}, err error) {
	rows, err := m.DB.Query(psql, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		var (
			val   interface{}
			slice reflect.Value
		)
		val, err = srf(rows)
		if err != nil {
			return
		}
		slice = reflect.Append(reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(val)), 0, InitialCapacity), reflect.ValueOf(val))

		for rows.Next() {
			val, err = srf(rows)
			if err != nil {
				return
			}
			slice = reflect.Append(slice, reflect.ValueOf(val))
		}
		ret = slice.Interface()
	}
	if err = rows.Err(); err != nil {
		return
	}
	return
}

func (m *clickhImpl) ScanAllAsync(psql string, srf ScanRowFunc, results chan interface{}, args ...interface{}) (err error) {
	rows, err := m.DB.Query(psql, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		var val interface{}
		val, err = srf(rows)
		if err != nil {
			return
		}

		results <- val

		for rows.Next() {
			val, err = srf(rows)
			if err != nil {
				return
			}

			results <- val
		}
	}
	if err = rows.Err(); err != nil {
		return
	}
	return
}

func (m *clickhImpl) ScanOne(psql string, srf ScanRowFunc, args ...interface{}) (ret interface{}, err error) {
	rows, err := m.DB.Query(psql, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	if rows.Next() {
		ret, err = srf(rows)
		if err != nil {
			return
		}
	}
	if err = rows.Err(); err != nil {
		return
	}

	return
}
