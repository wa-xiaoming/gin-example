package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func main() {
	dsn := flag.String("dsn", "", "consult[https://gorm.io/docs/connecting_to_the_database.html]")
	tables := flag.String("tables", "", "enter the required data table or leave it blank")
	flag.Parse()

	if *dsn == "" {
		log.Fatal("dsn cannot be empty, please provide a valid dsn value.")
	}

	// 链接数据库
	db, err := gorm.Open(mysql.Open(*dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		log.Fatalf("failed to connect database: %s", err.Error())
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory:%s", err.Error())
	}

	// 初始化代码生成器
	g := gen.NewGenerator(gen.Config{
		OutPath:      wd + "/internal/repository/mysql/dao",   // 指定生成代码的输出目录
		ModelPkgPath: wd + "/internal/repository/mysql/model", // 指定模型文件存放的包路径
		Mode:         gen.WithoutContext | gen.WithDefaultQuery,
	})

	// 使用已连接的数据库
	g.UseDB(db)

	// 生成表的模型
	tableMaps := strings.Split(*tables, ",")
	if len(tableMaps) == 0 || *tables == "" {
		g.ApplyBasic(
			g.GenerateAllTable()...,
		)
	} else {
		tableList := make([]string, 0, len(tableMaps))
		for _, tableName := range tableMaps {
			_tableName := strings.TrimSpace(string(tableName)) // trim leading and trailing space in tableName
			if _tableName == "" {                              // skip empty tableName
				continue
			}
			tableList = append(tableList, _tableName)
		}

		// Execute some data table tasks
		models := make([]interface{}, len(tableList))
		for i, tableName := range tableList {
			models[i] = g.GenerateModel(tableName)
		}
		g.ApplyBasic(models...)
	}

	// 生成代码
	g.Execute()
}
