package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gin-example/internal/pkg/stress"
)

func main() {
	// 定义命令行参数
	concurrency := flag.Int("c", 100, "并发数")
	totalRequests := flag.Int64("n", 10000, "总请求数")
	rateLimitRPS := flag.Int("r", 1000, "限流RPS")
	testDuration := flag.Duration("d", 60*time.Second, "测试持续时间")
	targetURL := flag.String("u", "/api/system/health", "目标URL")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// 创建压力测试配置
	config := &stress.StressTestConfig{
		Concurrency:   *concurrency,
		TotalRequests: *totalRequests,
		RateLimitRPS:  *rateLimitRPS,
		TestDuration:  *testDuration,
	}

	// 创建压力测试器
	tester := stress.NewStressTester(config)

	// 执行压力测试
	result := tester.Run(*targetURL)
	if result == nil {
		fmt.Println("压力测试执行失败")
		os.Exit(1)
	}

	// 输出结果到文件
	reportFile := fmt.Sprintf("stress_test_report_%d.txt", time.Now().Unix())
	f, err := os.Create(reportFile)
	if err != nil {
		fmt.Printf("创建报告文件失败: %v\n", err)
	} else {
		defer f.Close()
		f.WriteString(result.String())
		fmt.Printf("测试报告已保存到: %s\n", reportFile)
	}
}
