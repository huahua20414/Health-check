// Package common 定义项目共享的基础常量和运行时通用配置。
package common

import "github.com/milvus-io/milvus-sdk-go/v2/entity"

const (
	MilvusDBName         = "agent"
	MilvusCollectionName = "biz"

	MilvusFieldID       = "id"
	MilvusFieldVector   = "vector"
	MilvusFieldContent  = "content"
	MilvusFieldMetadata = "metadata"
	MilvusEmbeddingDim  = 1024
	MilvusVectorBitDim  = MilvusEmbeddingDim * 32
)

func MilvusFields() []*entity.Field {
	return []*entity.Field{
		{
			Name:     MilvusFieldID,
			DataType: entity.FieldTypeVarChar,
			TypeParams: map[string]string{
				"max_length": "256",
			},
			PrimaryKey: true,
		},
		{
			Name:     MilvusFieldVector,
			DataType: entity.FieldTypeBinaryVector,
			TypeParams: map[string]string{
				"dim": "32768",
			},
		},
		{
			Name:     MilvusFieldContent,
			DataType: entity.FieldTypeVarChar,
			TypeParams: map[string]string{
				"max_length": "8192",
			},
		},
		{
			Name:     MilvusFieldMetadata,
			DataType: entity.FieldTypeJSON,
		},
	}
}

var FileDir = "./docs/"
