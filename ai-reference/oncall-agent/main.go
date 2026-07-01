// Package main 启动 GoFrame HTTP 服务，并注册聊天、上传与运维分析接口。
package main

import (
	"oncall-agent/internal/ai/rag/syncer"
	"oncall-agent/internal/controller/chat"
	"oncall-agent/utility/common"
	"oncall-agent/utility/middleware"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.New()
	fileDir, err := g.Cfg().Get(ctx, "file_dir")
	if err != nil {
		panic(err)
	}
	common.FileDir = fileDir.String()
	if err := syncer.StartFromConfig(ctx); err != nil {
		g.Log().Warningf(ctx, "start rag syncer failed: %v", err)
	}
	s := g.Server()
	s.Group("/api", func(group *ghttp.RouterGroup) {
		group.Middleware(middleware.CORSMiddleware)
		group.Middleware(middleware.ResponseMiddleware)
		//这里需要一个ctrl对象注入
		group.Bind(chat.NewV1())
	})
	s.SetPort(6872)
	s.Run()
}
