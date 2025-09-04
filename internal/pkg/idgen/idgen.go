package idgen

import (
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

// initNode 初始化snowflake节点
func initNode() {
	var err error
	node, err = snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
}

// GenerateUniqueID 生成唯一ID
func GenerateUniqueID() string {
	once.Do(initNode)
	return node.Generate().String()
}