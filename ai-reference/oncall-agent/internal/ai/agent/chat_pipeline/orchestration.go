// Package chat_pipeline 中的本文件负责编排对话 Agent 的 RAG 与 ReAct 主流程。
package chat_pipeline

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// 添加输入转换节点 将用户输入转换一下给向量检索工具使用

// 添加输入转换节点 将用户输入转换成query history time
//添加模版节点,

//添加ReAct Agent节点,
//获取模型，配合四个工具创建一个react的agent实例
// agent的四个工具，查询服务器日志工具(mcp自动暴露)，grafana连接并查询所有的alert警告并返回给大模型能看懂的结构体字符串，
// mysql根据大模型输入自动查询数据并返回数据,获取当前时间,向量检索工具(由向量数据库和大模型组成，大模型用于将输入转换为向量)

//添加向量检索节点,由向量数据库和大模型组成,输出放在document字段里，下一步和输入转换节点配合连接template节点可以将输出直接放在模版字符串里

// 添加输出转换节点
func BuildChatAgent(ctx context.Context) (r compose.Runnable[*UserMessage, *schema.Message], err error) {
	const (
		InputToRag      = "InputToRag"
		ChatTemplate    = "ChatTemplate"
		ReactAgent      = "ReactAgent"
		MilvusRetriever = "MilvusRetriever"
		InputToChat     = "InputToChat"
	)
	//输入是用户输入，输出是 rag 查询结果
	g := compose.NewGraph[*UserMessage, *schema.Message]()
	//添加一个lambda节点，自定义执行函数,这里将query和history拼接起来，当然也有字符限制，用于rag查询，rag查询的时候会将这段文本解析出来，然后大模型生成query用于向量检索
	_ = g.AddLambdaNode(InputToRag, compose.InvokableLambdaWithOption(newInputToRagLambda), compose.WithNodeName("UserMessageToRag"))
	//获取模版
	chatTemplateKeyOfChatTemplate, err := newChatTemplate(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatTemplateNode(ChatTemplate, chatTemplateKeyOfChatTemplate)
	//创建 ReAct Agent 实例
	//将 Agent 包装为 Graph 可用的 Lambda 节点
	reactAgentKeyOfLambda, err := newReactAgentLambda(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLambdaNode(ReactAgent, reactAgentKeyOfLambda, compose.WithNodeName("ReActAgent"))
	//向量检索,这里有query改写然后粗排然后计算余弦相似度（粗排后的doc向量和query计算）
	milvusRetrieverKeyOfRetriever, err := newRetriever(ctx)
	if err != nil {
		return nil, err
	}
	// 注意下面的 output key 设置，把查询出来的设置为了documents，匹配 ChatTemplate 里面说prompt
	_ = g.AddRetrieverNode(MilvusRetriever, milvusRetrieverKeyOfRetriever, compose.WithOutputKey("documents"))
	_ = g.AddLambdaNode(InputToChat, compose.InvokableLambdaWithOption(newInputToChatLambda), compose.WithNodeName("UserMessageToChat"))
	_ = g.AddEdge(compose.START, InputToRag)
	_ = g.AddEdge(compose.START, InputToChat)
	_ = g.AddEdge(ReactAgent, compose.END)
	_ = g.AddEdge(InputToRag, MilvusRetriever)
	_ = g.AddEdge(MilvusRetriever, ChatTemplate)
	_ = g.AddEdge(InputToChat, ChatTemplate)
	_ = g.AddEdge(ChatTemplate, ReactAgent)
	r, err = g.Compile(ctx, compose.WithGraphName("ChatAgent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, err
}
