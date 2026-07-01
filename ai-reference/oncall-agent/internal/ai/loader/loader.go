// Package loader 封装知识库文件加载器，用于将本地文档读入处理流水线。
package loader

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino/components/document"
)

func NewFileLoader(ctx context.Context) (ldr document.Loader, err error) {
	config := &file.FileLoaderConfig{}
	ldr, err = file.NewFileLoader(ctx, config)
	if err != nil {
		return nil, err
	}
	return ldr, nil
}
