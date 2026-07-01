// Package main 提供知识库召回的命令行入口，用于验证向量检索效果。
package main

import (
	"context"
	"fmt"
	retriever2 "oncall-agent/internal/ai/retriever"
)

func main() {
	ctx := context.Background()
	r, err := retriever2.NewMilvusRetriever(ctx)
	if err != nil {
		panic(err)
	}
	query := "服务下线是什么原因"
	docs, err := r.Retrieve(ctx, query)
	if err != nil {
		panic(err)
	}
	fmt.Println("Q：", query)
	for _, doc := range docs {
		fmt.Println("A：", doc.Content)
	}
	fmt.Println("Done", len(docs))
}
