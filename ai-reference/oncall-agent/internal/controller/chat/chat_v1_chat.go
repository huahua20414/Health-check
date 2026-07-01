// Package chat 中的本文件实现普通对话接口，并维护会话记忆。
package chat

import (
	"context"

	"oncall-agent/api/chat/v1"
	"oncall-agent/internal/ai/agent/chat_pipeline"
	"oncall-agent/utility/log_call_back"
	"oncall-agent/utility/mem"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func (c *ControllerV1) Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error) {
	id := req.Id
	msg := req.Question
	// 读取历史时把当前问题一并传进去，这样 memory 层可以做“按 query 召回相关长期记忆”。
	history, err := mem.GetMessagesForQuery(ctx, id, msg)
	if err != nil {
		return nil, err
	}
	userMessage := &chat_pipeline.UserMessage{
		ID:      id,
		Query:   msg,
		History: history,
	}
	//构建一个图用来执行
	// 添加输入转换节点 将用户输入转换一下给向量检索工具使用
	//添加向量检索节点,由向量数据库和大模型组成,输出放在document字段里，下一步和输入转换节点配合连接template节点可以将输出直接放在模版字符串里

	// 添加输入转换节点 将用户输入转换成query history time

	//添加模版节点,

	//添加ReAct Agent节点,
	//模型，配合四个工具创建一个react的agent实例
	// agent的四个工具，查询服务器日志工具(mcp自动暴露)，grafana连接并查询所有的alert警告并返回给大模型能看懂的结构体字符串，
	// mysql根据大模型输入自动查询数据并返回数据,获取当前时间,向量检索工具(由向量数据库和大模型组成，大模型用于将输入转换为向量)

	//react就是最后一个节点用来输出
	runner, err := chat_pipeline.BuildChatAgent(ctx)
	if err != nil {
		return nil, err
	}
	//日志回调用于展示节点开始信息和名字以及节点的输入信息是什么,开启debug模式默认以有缩进的json字符串展示
	out, err := runner.Invoke(ctx, userMessage, compose.WithCallbacks(log_call_back.LogCallback(nil)))
	if err != nil {
		return nil, err
	}
	res = &v1.ChatRes{
		Answer: out.Content,
	}
	//将前端传过来的id设置一下问题和答案保存到history中
	if err := mem.AppendMessages(ctx, id,
		schema.UserMessage(msg),
		schema.AssistantMessage(out.Content, nil),
	); err != nil {
		return nil, err
	}

	return res, nil
}
