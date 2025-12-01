package model

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v7"
	"gorm.io/gorm"
)

// ISQLLoader loader for dict results
type ISQLLoader interface {
	LoadDictData(map[string]interface{})
}

// rowsClose When scan is not finished or scan operation is not performed, need to manually release the connection
func rowsClose(rows *sql.Rows) {
	if rows != nil {
		rows.Close()
	}
}

// clearGormTransaction gorm transaction cleanup
// Usage:
// tx := gormDB.Begin()
// defer clearGormTransaction(tx)
func clearGormTransaction(tx *gorm.DB) {
	if err := recover(); err != nil {
		tx.Rollback()
	}
}

func clearTransaction(tx *sql.Tx) {
	err := tx.Rollback()
	if err != sql.ErrTxDone && err != nil {
		fmt.Println("error in transaction", err)
	}
}

// dataGetByRows Get data from database return results, parse into dict form
func dataGetByRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	size := len(columns)
	var obj []map[string]interface{}

	colData := make([]interface{}, size)
	container := make([]interface{}, size)
	for i := range colData {
		colData[i] = &container[i]
	}

	for rows.Next() {
		err := rows.Scan(colData...)
		if err != nil {
			return nil, err
		}
		var r = make(map[string]interface{}, size)
		for i, column := range columns {
			r[column] = colData[i]
		}

		obj = append(obj, r)
	}

	return obj, nil
}

func rSet(redisDB *redis.Client, bucket string, key string, value interface{}) error {
	p, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return redisDB.HSet(bucket, key, p).Err()
}
func rDel(redisDB *redis.Client, bucket, key string) error {
	return redisDB.HDel(bucket, key).Err()
}

func rGet(redisDB *redis.Client, bucket string, key string, dest interface{}) error {
	p, err := redisDB.HGet(bucket, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(p), dest)
}
