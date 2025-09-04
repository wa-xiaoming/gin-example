package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type TemplateData struct {
	PackageName  string
	VariableName string
	StructName   string
}

func main() {
	table := flag.String("table", "", "enter the required data table")
	flag.Parse()

	tableName := *table
	if tableName == "" {
		log.Fatal("table cannot be empty, please provide a valid table name.")
	}

	// 准备模板数据
	data := TemplateData{
		PackageName:  tableName,
		VariableName: tableName,
		StructName:   toCamelCase(tableName),
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory:%s", err.Error())
	}

	// handler 模板文件路径
	handlerTmpl, err := template.ParseFiles(fmt.Sprintf("%s/cmd/handlergen/handler_template.go.tpl", wd))
	if err != nil {
		log.Fatal(err)
	}

	// routers 模板文件路径
	routersTmpl, err := template.ParseFiles(fmt.Sprintf("%s/cmd/handlergen/routers_template.go.tpl", wd))
	if err != nil {
		log.Fatal(err)
	}

	// 创建输出目录
	outputDir := fmt.Sprintf("%s/internal/api/%s", wd, tableName)
	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	// 创建 Handlers 文件
	outputHandlerFile := filepath.Join(outputDir, fmt.Sprintf("%s_handler.gen.go", tableName))
	handlerFile, err := os.Create(outputHandlerFile)
	if err != nil {
		log.Fatal(err)
	}
	defer handlerFile.Close()

	err = handlerTmpl.Execute(handlerFile, data)
	if err != nil {
		log.Fatal(err)
	}

	// 创建 Routers 文件
	outputRoutersFile := filepath.Join(outputDir, fmt.Sprintf("%s_routers.gen.go", tableName))
	routersFile, err := os.Create(outputRoutersFile)
	if err != nil {
		log.Fatal(err)
	}
	defer routersFile.Close()

	err = routersTmpl.Execute(routersFile, data)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Handler File created: %s", outputHandlerFile)
	log.Printf("Routers File created: %s", outputRoutersFile)
	log.Println("Template execution completed successfully.")
}

// 将字符串转为驼峰式命名
func toCamelCase(s string) string {
	// 用下划线分割字符串
	parts := strings.Split(s, "_")

	// 对每个部分首字母大写
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.Title(parts[i])
	}

	// 拼接所有部分
	return strings.Join(parts, "")
}
