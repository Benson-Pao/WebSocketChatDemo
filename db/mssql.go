package db

import (
	fileIo "IOCopyByWS/io"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/denisenkom/go-mssqldb"
)

var (
	connString = ""
	dbFile     = "./data.cfg"
)

func LoadConnectFile() error {
	connBytes, err := fileIo.Read(dbFile)
	if err != nil {
		log.Println(err)
		return err
	}
	SetConnectString(string(connBytes))
	return nil
}

func SetConnectString(connectString string) {
	connString = connectString
}

func ExecSP(procName string, parms []interface{}) (interface{}, error) {
	db, err := sql.Open("mssql", connString)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer db.Close()

	result := []map[string]interface{}{}

	// 執行存儲過程並獲取結果

	rows, err := db.Query(fmt.Sprintf("EXEC %s", procName), parms...)
	if err != nil {
		log.Println(err, fmt.Sprintf(":EXEC %s %v", procName, parms))
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	// 遍歷結果集
	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		rowValues := map[string]interface{}{}
		for i, value := range values {
			columnName := columns[i]
			rowValues[columnName] = value
		}

		result = append(result, rowValues)
	}

	return result, nil
}
