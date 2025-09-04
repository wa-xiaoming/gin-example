package env

import (
	"fmt"
	"os"
	"strings"
)

var (
	active Environment
	dev    Environment = &environment{value: "dev"}
	fat    Environment = &environment{value: "fat"}
	uat    Environment = &environment{value: "uat"}
	pro    Environment = &environment{value: "pro"}
)

var _ Environment = (*environment)(nil)

// Environment 环境配置
type Environment interface {
	Value() string
	IsDev() bool
	IsFat() bool
	IsUat() bool
	IsPro() bool
}

type environment struct {
	value string
}

func (e *environment) Value() string {
	return e.value
}

func (e *environment) IsDev() bool {
	return e.value == "dev"
}

func (e *environment) IsFat() bool {
	return e.value == "fat"
}

func (e *environment) IsUat() bool {
	return e.value == "uat"
}

func (e *environment) IsPro() bool {
	return e.value == "pro"
}

// SetEnv 设置环境变量（用于测试或程序内部设置）
func SetEnv(env string) {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "dev":
		active = dev
	case "fat":
		active = fat
	case "uat":
		active = uat
	case "pro":
		active = pro
	default:
		active = fat
	}
}

func init() {
	// 从环境变量或命令行参数获取环境设置
	env := getEnvFromOS()

	switch strings.ToLower(strings.TrimSpace(env)) {
	case "dev":
		active = dev
	case "fat":
		active = fat
	case "uat":
		active = uat
	case "pro":
		active = pro
	default:
		active = fat
		fmt.Println("Warning: 'env' cannot be found, or it is illegal. The default 'fat' will be used.")
	}
}

// getEnvFromOS 从环境变量或命令行参数获取环境设置
func getEnvFromOS() string {
	// 首先检查环境变量
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}
	
	// 然后检查命令行参数
	for i, arg := range os.Args {
		if arg == "-env" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, "-env=") {
			return strings.TrimPrefix(arg, "-env=")
		}
	}
	
	return "fat" // 默认环境
}

// Active 当前配置的env
func Active() Environment {
	return active
}