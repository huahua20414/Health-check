import React, { useMemo, useState } from 'react'
import { Bot, Sparkles } from 'lucide-react'
import { Button, Card, Empty, PageHeader, Textarea } from '../components/UI.jsx'
import { useHealth } from '../HealthContext.jsx'

const quickPrompts = [
  '这个项目的预约链路是怎么闭环的？',
  '管理员、医生、用户三种角色的权限边界是什么？',
  '医生号源、排班模板、机构绑定套餐之间是什么关系？',
  '家庭成员、预约单、报告、评价之间的数据关系怎么解释？',
]

export function AIAssistantView() {
	const h = useHealth()
	const [question, setQuestion] = useState('')
	const [answer, setAnswer] = useState(null)
	const canAsk = useMemo(() => question.trim().length > 0, [question])

	async function submit() {
		if (!canAsk) return
		try {
			const result = await h.askAI(question)
			setAnswer(result)
		} catch (error) {
			h.notify('error', error.message)
		}
	}

	return (
		<>
			<PageHeader title="AI 助手" subtitle="结合当前项目规则、数据字典和复制进来的 AI 设计资料做问答。" />
			<div className="ai-assistant-layout">
				<Card className="ai-assistant-main" title="对话输入" subtitle="适合问业务规则、数据库关系、权限边界、排班与预约逻辑。" actions={<Button onClick={submit} loading={h.loading.aiAssistant} disabled={!canAsk}>发送问题</Button>}>
					<div className="ai-hero">
						<div className="ai-hero-icon"><Bot size={24} /></div>
						<div>
							<strong>项目内 AI 知识助手</strong>
							<p>优先检索当前项目文档和引入的 oncall-agent 设计资料；如果配置了模型，会再基于命中文档生成更自然的回答。</p>
						</div>
					</div>
					<Textarea className="ai-question-box" value={question} onChange={(event) => setQuestion(event.target.value)} placeholder="例如：为什么新增机构后不应该自动补没有绑定医生的号源？" />
					<div className="ai-quick-prompts">
						{quickPrompts.map((item) => <button key={item} type="button" className="ai-quick-prompt" onClick={() => setQuestion(item)}>{item}</button>)}
					</div>
				</Card>
				<Card className="ai-assistant-side" title="能力范围" subtitle="当前不接服务器日志、告警、MCP。">
					<div className="stack-list ai-capability-list">
						<div className="accent-row"><div><strong>已接入</strong><span>项目文档检索、数据库数据字典、权限与预约规则问答、AI 设计资料参考</span></div><Sparkles size={16} /></div>
						<div className="accent-row"><div><strong>未接入</strong><span>日志排障、告警分析、Grafana、MCP 工具链</span></div><Sparkles size={16} /></div>
					</div>
				</Card>
			</div>
			<Card title="回答结果" subtitle={answer ? `回答模式：${answer.mode || '-'}${answer.usedModel ? ' · 已调用模型' : ' · 仅检索资料'}` : '提交问题后会在这里展示。'}>
				{answer ? (
					<div className="ai-answer-block">
						<div className="ai-answer-text">{answer.answer}</div>
						<div className="ai-citation-list">
							<h4>参考资料</h4>
							{answer.citations?.length ? answer.citations.map((item, index) => (
								<div key={`${item.source}-${index}`} className="ai-citation-item">
									<strong>{item.title}</strong>
									<small>{item.source}</small>
									<p>{item.snippet}</p>
								</div>
							)) : <Empty text="本次没有返回参考资料" />}
						</div>
					</div>
				) : <Empty text="还没有提问" />}
			</Card>
		</>
	)
}

