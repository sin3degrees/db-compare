package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"db-compare/conf"

	_ "github.com/go-sql-driver/mysql"
)

var (
	db1Name = conf.Sysconfig.DBSrc.Database
	db1User = conf.Sysconfig.DBSrc.User
	db1Pass = conf.Sysconfig.DBSrc.Password
	db1Host = conf.Sysconfig.DBSrc.Host + ":" + conf.Sysconfig.DBSrc.Port

	db2Name = conf.Sysconfig.DBDst.Database
	db2User = conf.Sysconfig.DBDst.User
	db2Pass = conf.Sysconfig.DBDst.Password
	db2Host = conf.Sysconfig.DBDst.Host + ":" + conf.Sysconfig.DBDst.Port

	tbOnly      = conf.Sysconfig.TbOnly
	tbIgnore    = conf.Sysconfig.TbIgnore
	tbOnlyStr   = strings.Join(tbOnly, ",")
	tbIgnoreStr = strings.Join(tbIgnore, ",")
)

func main() {
	// Connect to database1
	db1, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", db1User, db1Pass, db1Host, db1Name))
	if err != nil {
		panic(err)
	}
	defer db1.Close()

	// Connect to database2
	db2, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", db2User, db2Pass, db2Host, db2Name))
	if err != nil {
		panic(err)
	}
	defer db2.Close()

	// Get list of tables from database1
	rows, err := db1.Query("SHOW TABLES")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	// Get list of tables from database2
	tableMap := make(map[string]bool)
	rows2, err := db2.Query("SHOW TABLES")
	if err != nil {
		panic(err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var tableName string
		err := rows2.Scan(&tableName)
		if err != nil {
			panic(err)
		}
		tableMap[tableName] = true
	}

	// 打开文件，如果文件不存在则创建，如果文件存在则以追加方式打开
	file, err := os.OpenFile("result.sql", os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	// Iterate over tables and compare with database2
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		if err != nil {
			panic(err)
		}
		if len(tbOnly) > 0 && !strings.Contains(tbOnlyStr, tableName) {
			continue
		}
		if len(tbIgnore) > 0 && strings.Contains(tbIgnoreStr, tableName) {
			continue
		}

		// Get list of columns from table in database1
		colRows, err := db1.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", tableName))
		if err != nil {
			panic(err)
		}
		defer colRows.Close()

		if tableMap[tableName] {
			// Get list of columns from table in database2
			colMap := make(map[string]bool)
			colMap["id"] = true // Assume there is an "id" column
			colRows2, err := db2.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", tableName))
			if err != nil {
				panic(err)
			}
			defer colRows2.Close()
			for colRows2.Next() {
				var colName string
				var colType string
				var colNull string
				var colKey string
				var colDefault interface{}
				var colExtra string
				err := colRows2.Scan(&colName, &colType, &colNull, &colKey, &colDefault, &colExtra)
				if err != nil {
					panic(err)
				}
				colMap[colName] = true
			}

			// Compare columns
			var newCols []string
			for colRows.Next() {
				var colName string
				var colType string
				var colNull string
				var colKey string
				var colDefault interface{}
				var colExtra string
				err := colRows.Scan(&colName, &colType, &colNull, &colKey, &colDefault, &colExtra)
				if err != nil {
					panic(err)
				}
				if !colMap[colName] {
					newCols = append(newCols, fmt.Sprintf("\nADD COLUMN %s %s %s%s%s", colName, colType, map[bool]string{true: "NULL", false: "NOT NULL"}[colNull == "YES"],
						map[bool]string{true: fmt.Sprintf(" DEFAULT %v", colDefault), false: ""}[colDefault != nil],
						map[bool]string{true: " " + strings.ToUpper(colExtra), false: ""}[colExtra != ""]))
				}
			}

			// Generate SQL statement
			if len(newCols) > 0 {
				sql := fmt.Sprintf("ALTER TABLE %s %s;", tableName, strings.Join(newCols, ", "))
				fmt.Println(sql)
				// 写入字符串
				_, err = file.WriteString(sql + "\n\n")
				if err != nil {
					fmt.Println(err)
					return
				}
			}
		} else {
			// Generate SQL statement
			var cols []string
			var priStr string = ""
			for colRows.Next() {
				var colName string
				var colType string
				var colNull string
				var colKey string
				var colDefault interface{}
				var colExtra string
				err := colRows.Scan(&colName, &colType, &colNull, &colKey, &colDefault, &colExtra)
				if err != nil {
					panic(err)
				}
				if colKey == "PRI" {
					priStr = fmt.Sprintf("PRIMARY KEY (%s) USING BTREE", colName)
				}
				cols = append(cols, fmt.Sprintf("\n%s %s %s%s%s", colName, colType, map[bool]string{true: "NULL", false: "NOT NULL"}[colNull == "YES"],
					map[bool]string{true: fmt.Sprintf(" DEFAULT %v", colDefault), false: ""}[colDefault != nil],
					map[bool]string{true: " " + strings.ToUpper(colExtra), false: ""}[colExtra != ""]))
			}
			sql := fmt.Sprintf("CREATE TABLE %s (%s) \n%s;", tableName, strings.Join(cols, ", "), priStr)
			fmt.Println(sql)
			// 写入字符串
			_, err = file.WriteString(sql + "\n\n")
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}
