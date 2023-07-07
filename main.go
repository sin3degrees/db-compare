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
	tbOnlyStr   = strings.Join(tbOnly, ",") + ","   // Add comma to end of string to make sure we don't match partial table names
	tbIgnoreStr = strings.Join(tbIgnore, ",") + "," // Add comma to end of string to make sure we don't match partial table names
)

type colInfo struct {
	Field      string
	Type       string
	Collation  interface{}
	Null       string
	Key        string
	Default    interface{}
	Extra      string
	Privileges string
	Comment    string
}

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
	rows, err := db1.Query("SHOW FULL TABLES")
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
		var tableType string
		err := rows.Scan(&tableName, &tableType)
		if err != nil {
			panic(err)
		}
		if len(tbOnly) > 0 && !strings.Contains(tbOnlyStr, tableName+",") {
			continue
		}
		if len(tbIgnore) > 0 && strings.Contains(tbIgnoreStr, tableName+",") {
			continue
		}

		if tableType == "VIEW" {
			viewRows, err := db1.Query(fmt.Sprintf("SELECT VIEW_DEFINITION FROM INFORMATION_SCHEMA.VIEWS WHERE TABLE_NAME = '%s'", tableName))
			if err != nil {
				panic(err)
			}
			defer viewRows.Close()
			var viewDef string
			for viewRows.Next() {
				err := viewRows.Scan(&viewDef)
				if err != nil {
					panic(err)
				}
			}
			viewDef = strings.ReplaceAll(viewDef, "`"+conf.Sysconfig.DBSrc.Database+"`.", "")
			if !tableMap[tableName] {
				sql := fmt.Sprintf("CREATE VIEW %s AS %s;", tableName, viewDef)
				_, err = file.WriteString(sql + "\n\n")
				if err != nil {
					fmt.Println(err)
					return
				}
			} else {
				viewRows2, err := db2.Query(fmt.Sprintf("SELECT VIEW_DEFINITION FROM INFORMATION_SCHEMA.VIEWS WHERE TABLE_NAME = '%s'", tableName))
				if err != nil {
					panic(err)
				}
				defer viewRows2.Close()
				var viewDef2 string
				for viewRows2.Next() {
					err := viewRows2.Scan(&viewDef2)
					if err != nil {
						panic(err)
					}
				}
				viewDef2 = strings.ReplaceAll(viewDef2, "`"+conf.Sysconfig.DBDst.Database+"`.", "")
				if viewDef != viewDef2 {
					sql := fmt.Sprintf("ALTER VIEW %s AS %s;", tableName, viewDef)
					_, err = file.WriteString(sql + "\n\n")
					if err != nil {
						fmt.Println(err)
						return
					}
				}
			}
			continue
		}
		// Get list of columns from table in database1
		colRows, err := db1.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM %s", tableName))
		if err != nil {
			panic(err)
		}
		defer colRows.Close()

		if tableMap[tableName] {
			// Get list of columns from table in database2
			colMap := make(map[string]bool)
			colMap["id"] = true // Assume there is an "id" column
			colRows2, err := db2.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM %s", tableName))
			if err != nil {
				panic(err)
			}
			defer colRows2.Close()
			for colRows2.Next() {
				var colInfo = colInfo{}
				err := colRows2.Scan(&colInfo.Field, &colInfo.Type, &colInfo.Collation, &colInfo.Null, &colInfo.Key,
					&colInfo.Default, &colInfo.Extra, &colInfo.Privileges, &colInfo.Comment)
				if err != nil {
					panic(err)
				}
				colMap[colInfo.Field] = true
			}

			// Compare columns
			var newCols []string
			for colRows.Next() {
				var colInfo = colInfo{}
				err := colRows.Scan(&colInfo.Field, &colInfo.Type, &colInfo.Collation, &colInfo.Null, &colInfo.Key,
					&colInfo.Default, &colInfo.Extra, &colInfo.Privileges, &colInfo.Comment)
				if err != nil {
					panic(err)
				}
				if !colMap[colInfo.Field] {
					newCols = append(newCols, fmt.Sprintf("\nADD COLUMN %s %s %s%s%s COMMENT '%s'", colInfo.Field, colInfo.Type,
						map[bool]string{true: "NULL", false: "NOT NULL"}[colInfo.Null == "YES"],
						map[bool]string{true: fmt.Sprintf(" DEFAULT %s", colInfo.Default), false: ""}[colInfo.Default != nil],
						map[bool]string{true: " " + strings.ToUpper(colInfo.Extra), false: ""}[colInfo.Extra != ""], colInfo.Comment))
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
				var colInfo = colInfo{}
				err := colRows.Scan(&colInfo.Field, &colInfo.Type, &colInfo.Collation, &colInfo.Null, &colInfo.Key,
					&colInfo.Default, &colInfo.Extra, &colInfo.Privileges, &colInfo.Comment)
				if err != nil {
					panic(err)
				}
				if colInfo.Key == "PRI" {
					priStr = fmt.Sprintf("PRIMARY KEY (%s) USING BTREE", colInfo.Field)
				}
				cols = append(cols, fmt.Sprintf("\n%s %s %s%s%s COMMENT '%s'", colInfo.Field, colInfo.Type,
					map[bool]string{true: "NULL", false: "NOT NULL"}[colInfo.Null == "YES"],
					map[bool]string{true: fmt.Sprintf(" DEFAULT %s", colInfo.Default), false: ""}[colInfo.Default != nil],
					map[bool]string{true: " " + strings.ToUpper(colInfo.Extra), false: ""}[colInfo.Extra != ""], colInfo.Comment))
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
