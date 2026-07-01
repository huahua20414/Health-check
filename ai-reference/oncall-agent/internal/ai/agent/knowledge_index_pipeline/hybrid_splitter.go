// Package knowledge_index_pipeline 中的本文件实现自定义混合分片器。
package knowledge_index_pipeline

import (
	// context 只是为了满足 Transformer 接口签名。
	"context"
	// fmt 用来拼接 chunk ID。
	"fmt"
	// regexp 用来识别 Markdown 标题。
	"regexp"
	// strings 提供文本清洗、拼接和切分能力。
	"strings"

	// document 提供文档转换器接口。
	"github.com/cloudwego/eino/components/document"
	// schema 提供文档结构体定义。
	"github.com/cloudwego/eino/schema"
	// uuid 用于在原文档没有 ID 时生成唯一前缀。
	"github.com/google/uuid"
)

// chunkingStrategyName 记录当前采用的分片策略名，方便排查和分析。
const chunkingStrategyName = "semantic_section+sliding_window_overlap"

// markdownHeadingPattern 用来识别 Markdown 标题行。
// 第 1 个捕获组是 # 的数量，第 2 个捕获组是真正的标题文本。
var markdownHeadingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)

// hybridDocumentSplitter 是这次自定义的混合分片器。
// 它不是单纯按标题切，也不是单纯固定长度切，而是把两种策略结合起来：
// 1. 先按 Markdown 结构做“语义分段”；
// 2. 再对每个分段做“固定窗口 + overlap”切块。
type hybridDocumentSplitter struct {
	// cfg 保存当前 splitter 的切分参数。
	cfg *chunkingConfig
}

// semanticSection 表示一次“语义分段”后的中间结果。
// 后面真正切 chunk 时，会以 section 为边界，尽量不跨大标题乱切。
type semanticSection struct {
	// Title 是当前 section 的标题文本。
	Title string
	// Level 是标题级别，比如 # 是 1，## 是 2。
	Level int
	// Content 是当前 section 的完整内容。
	Content string
}

func newHybridDocumentSplitter(cfg *chunkingConfig) document.Transformer {
	// 返回实现了 document.Transformer 接口的具体实例。
	return &hybridDocumentSplitter{cfg: cfg}
}

func (h *hybridDocumentSplitter) Transform(_ context.Context, docs []*schema.Document, _ ...document.TransformerOption) ([]*schema.Document, error) {
	// result 用来收集所有输入文档切分后的 chunk 文档。
	result := make([]*schema.Document, 0, len(docs))
	// 依次处理每一篇输入文档。
	for _, doc := range docs {
		// 空指针文档直接跳过，避免 panic。
		if doc == nil {
			continue
		}

		// 第一步先把整篇文档拆成语义 section。
		sections := splitIntoSemanticSections(doc.Content)
		// chunkIndex 在当前文档内部递增，用于标记 chunk 顺序。
		chunkIndex := 0
		// 依次处理每一个语义 section。
		for _, section := range sections {
			// 第二步再把每个 section 切成更适合 embedding / 检索的 chunk。
			chunks := h.chunkSection(section)
			// 依次处理 section 中得到的每一个 chunk。
			for _, chunk := range chunks {
				// 空白 chunk 没有意义，直接跳过。
				if strings.TrimSpace(chunk) == "" {
					continue
				}
				// 这里把 chunk 策略、标题层级、chunk 序号等信息写进 metadata，
				// 方便后续做检索调优、问题排查或二次排序。
				// 先复制原文档的 metadata，避免多个 chunk 共用同一个 map。
				meta := cloneMeta(doc.MetaData)
				// 记录 chunk 在原文档里的顺序。
				meta["_chunk_index"] = chunkIndex
				// 记录 chunk 的字符长度。
				meta["_chunk_size"] = len([]rune(chunk))
				// 记录 chunk 使用的切分策略名。
				meta["_chunk_strategy"] = chunkingStrategyName
				// 记录 chunk 所属语义 section 的标题。
				meta["_semantic_title"] = section.Title
				// 记录 chunk 所属语义 section 的标题层级。
				meta["_semantic_level"] = section.Level
				// 把当前 chunk 封装成新的文档对象加入结果集。
				result = append(result, &schema.Document{
					// 生成当前 chunk 的唯一 ID。
					ID: buildChunkID(doc.ID, chunkIndex),
					// 写入 chunk 的正文内容。
					Content: chunk,
					// 写入补充好的 metadata。
					MetaData: meta,
				})
				// 每处理完一个 chunk，序号递增。
				chunkIndex++
			}
		}
	}
	// 返回切分完成后的所有 chunk 文档。
	return result, nil
}

func buildChunkID(originalID string, chunkIndex int) string {
	// 如果原始文档没有 ID，就临时生成一个 uuid 前缀。
	if originalID == "" {
		return fmt.Sprintf("%s-%d", uuid.New().String(), chunkIndex)
	}
	// 如果原始文档有 ID，就在后面拼上 chunk 序号。
	return fmt.Sprintf("%s-%d", originalID, chunkIndex)
}

func splitIntoSemanticSections(content string) []semanticSection {
	// 先统一换行和首尾空白，减少后续判断分支。
	content = normalizeContent(content)
	// 归一化后为空，说明没有可切分内容。
	if content == "" {
		return nil
	}

	// 逐行扫描文本，识别标题、空行和代码块边界。
	lines := strings.Split(content, "\n")
	// sections 保存最终切出来的语义段。
	sections := make([]semanticSection, 0, 8)
	// currentTitle 保存当前 section 的标题文本。
	currentTitle := ""
	// currentLevel 保存当前 section 的标题级别。
	currentLevel := 0
	// currentLines 累积当前 section 的正文行。
	currentLines := make([]string, 0, 32)
	// inCodeBlock 标记当前是否处于 fenced code block 中。
	inCodeBlock := false

	// flush 把 currentLines 累积的内容落成一个 section。
	flush := func() {
		// 每遇到一个新的标题，就把前面累积的 section 落下来。
		// 把当前 section 的各行用换行拼回完整文本。
		text := strings.TrimSpace(strings.Join(currentLines, "\n"))
		// 如果拼出来是空白内容，就只清空缓存，不生成 section。
		if text == "" {
			currentLines = currentLines[:0]
			return
		}
		// 把当前 section 加入结果集。
		sections = append(sections, semanticSection{
			// 记录当前 section 标题。
			Title: currentTitle,
			// 记录当前 section 标题层级。
			Level: currentLevel,
			// 记录当前 section 的正文内容。
			Content: text,
		})
		// 清空当前行缓存，准备收集下一段。
		currentLines = currentLines[:0]
	}
	// 循环处理文档中的每一行。
	for _, rawLine := range lines {
		// 去掉行尾多余空格，但保留行首缩进，避免破坏代码格式。
		line := strings.TrimRight(rawLine, " \t")
		// trimmed 用于做“是否为空行”“是否是标题”等判断。
		trimmed := strings.TrimSpace(line)

		// 如果当前行是 fenced code block 的起止标记，就切换代码块状态。
		if isCodeFence(trimmed) {
			// 代码块单独处理，避免把代码里的 #、空行误判成 Markdown 标题或段落边界。
			inCodeBlock = !inCodeBlock
			// 无论是开始还是结束标记，都保留到正文里。
			currentLines = append(currentLines, line)
			continue
		}

		// 只有在代码块外，才允许把内容识别成标题或空行。
		if !inCodeBlock {
			// 尝试把当前行匹配成 Markdown 标题。
			if matches := markdownHeadingPattern.FindStringSubmatch(trimmed); matches != nil {
				// 标题命中后，说明一个新的语义段落开始了。
				// 先把前一个 section 刷出去。
				flush()
				// 标题文本位于第 2 个捕获组。
				currentTitle = matches[2]
				// 标题级别由 # 的个数决定。
				currentLevel = len(matches[1])
				// 当前标题行本身也保留在 section 正文中。
				currentLines = append(currentLines, line)
				continue
			}

			// 空行可以作为段落边界信号。
			if trimmed == "" {
				// 连续空行不需要重复保留，但保留一个空行可以帮助后续按段落切分。
				// 只有前一行不是空行时，才追加一个空行占位。
				if len(currentLines) > 0 && strings.TrimSpace(currentLines[len(currentLines)-1]) != "" {
					currentLines = append(currentLines, "")
				}
				continue
			}
		}

		// 普通正文行直接加入当前 section。
		currentLines = append(currentLines, line)
	}
	// 循环结束后，把最后一个 section 也刷出去。
	flush()

	// 如果一个标题都没识别出来，就把整篇文档当成一个 section。
	if len(sections) == 0 {
		// 没有 Markdown 标题时，整篇文档就作为一个大 section 继续走后续切块。
		return []semanticSection{{Content: content}}
	}
	// 返回按标题切好的语义 section 列表。
	return sections
}

func (h *hybridDocumentSplitter) chunkSection(section semanticSection) []string {
	// 先按自然段切成 unit，尽量让 chunk 以段落为单位拼装，而不是硬按字符截断。
	// 一个 unit 可以理解成“构建 chunk 的最小拼装单元”。
	units := splitSectionIntoUnits(section.Content)
	// 如果 section 没有切出任何 unit，就没有 chunk 可返回。
	if len(units) == 0 {
		return nil
	}

	// chunks 收集当前 section 切出来的所有 chunk。
	chunks := make([]string, 0, len(units))
	// current 表示当前正在累积的 chunk 内容。
	var current string
	// 按顺序遍历每个 unit。
	for _, unit := range units {
		// 去掉 unit 首尾空白，避免生成脏 chunk。
		unit = strings.TrimSpace(unit)
		// 空 unit 不参与拼装。
		if unit == "" {
			continue
		}

		// 如果一个 unit 自己就超过最大长度，就不能整体塞进 chunk。
		if runeLen(unit) > h.cfg.MaxChunkSize {
			// 单段本身过长时，退回到滑动窗口切分。
			// 这是固定窗口策略的兜底，避免超长段落完全无法入库。
			// 如果 current 里已经有累积内容，先把它落成一个 chunk。
			if strings.TrimSpace(current) != "" {
				chunks = append(chunks, strings.TrimSpace(current))
				current = ""
			}
			// 超长 unit 自己走滑窗切分，并把结果追加到 chunks。
			chunks = append(chunks, splitLongText(unit, h.cfg.MaxChunkSize, h.cfg.ChunkOverlap)...)
			continue
		}

		// 先假设把当前 unit 拼到 current 后面。
		candidate := joinChunkParts(current, unit)
		// 如果 current 还为空，或者拼进去以后仍未超长，就继续累积。
		if current == "" || runeLen(candidate) <= h.cfg.MaxChunkSize {
			current = candidate
			continue
		}

		// 当前 chunk 已接近上限，就先落一个 chunk，
		// 再把 overlap 尾巴和下一个 unit 拼到新 chunk，保证边界附近上下文不丢。
		// 先把当前累积好的 chunk 放进结果里。
		chunks = append(chunks, strings.TrimSpace(current))
		// 新 chunk 由“上一个 chunk 的尾巴 + 当前 unit”组成。
		current = joinChunkParts(overlapTail(current, h.cfg.ChunkOverlap, h.cfg.MinChunkSize), unit)
		// 如果拼上 overlap 后仍然过长，就再退回滑窗切分。
		if runeLen(current) > h.cfg.MaxChunkSize {
			chunks = append(chunks, splitLongText(current, h.cfg.MaxChunkSize, h.cfg.ChunkOverlap)...)
			current = ""
		}
	}

	// 循环结束后，如果 current 还有剩余内容，也要落成 chunk。
	if strings.TrimSpace(current) != "" {
		chunks = append(chunks, strings.TrimSpace(current))
	}
	// 最后再做一次小块合并，避免分片太碎。
	return cleanupChunks(chunks, h.cfg.MinChunkSize)
}

func splitSectionIntoUnits(content string) []string {
	// 优先按空行切段，这通常比直接按句号切更稳，
	// 因为技术文档里一段通常就是一个相对完整的语义单元。
	// 双换行在这里被当成“段落分隔符”。
	parts := strings.Split(content, "\n\n")
	// units 保存清洗后的非空段落。
	units := make([]string, 0, len(parts))
	// 遍历每个按空行切出来的段落。
	for _, part := range parts {
		// 去掉段落首尾空白。
		part = strings.TrimSpace(part)
		// 空段落跳过。
		if part == "" {
			continue
		}
		// 非空段落加入 units。
		units = append(units, part)
	}
	// 正常切出段落时，直接返回。
	if len(units) > 0 {
		return units
	}
	// 如果没有双换行，就把整段内容当成一个 unit。
	return []string{content}
}

func splitLongText(text string, maxSize, overlap int) []string {
	// 这是典型的滑动窗口切分：
	// 每次取 maxSize，下一块从 end-overlap 开始。
	// 先把文本清洗后转成 rune，避免中文按字节切坏。
	runes := []rune(strings.TrimSpace(text))
	// 空文本没有 chunk 可切。
	if len(runes) == 0 {
		return nil
	}
	// 如果本来就不超长，直接原样返回。
	if len(runes) <= maxSize {
		return []string{string(runes)}
	}
	// overlap 不能大于等于窗口大小，否则窗口不会真正向前推进。
	if overlap >= maxSize {
		overlap = maxSize / 5
	}
	// overlap 也不能小于 0。
	if overlap < 0 {
		overlap = 0
	}

	// 预估容量，减少切片扩容次数。
	chunks := make([]string, 0, (len(runes)/maxSize)+1)
	// start 表示当前窗口的起始位置。
	start := 0
	// 只要还有剩余文本，就继续切下一个窗口。
	for start < len(runes) {
		// 默认窗口终点是 start + maxSize。
		end := start + maxSize
		// 最后一块不足 maxSize 时，终点截到文本末尾。
		if end > len(runes) {
			end = len(runes)
		}
		// 截出当前窗口文本并加入结果。
		chunks = append(chunks, strings.TrimSpace(string(runes[start:end])))
		// 如果已经切到文本末尾，就结束循环。
		if end == len(runes) {
			break
		}
		// 下一块从“当前终点往回退 overlap”开始，形成重叠区域。
		start = end - overlap
		// 理论保护：防止下标出现负数。
		if start < 0 {
			start = 0
		}
	}
	// 返回滑窗切好的所有块。
	return chunks
}

func cleanupChunks(chunks []string, minSize int) []string {
	// 如果切出来的小块过短，就尝试和前一个块合并，减少无效短 chunk。
	// 少于等于 1 个 chunk 时没有合并空间，直接返回。
	if len(chunks) <= 1 {
		return chunks
	}

	// result 保存清洗并按需合并后的 chunk。
	result := make([]string, 0, len(chunks))
	// 顺序处理每个 chunk。
	for _, chunk := range chunks {
		// 去掉 chunk 首尾空白。
		chunk = strings.TrimSpace(chunk)
		// 空 chunk 直接跳过。
		if chunk == "" {
			continue
		}

		// 第一个有效 chunk 直接放进结果。
		if len(result) == 0 {
			result = append(result, chunk)
			continue
		}

		// 如果当前 chunk 太短，并且合并后长度还可接受，就并到前一个 chunk 后面。
		if runeLen(chunk) < minSize && runeLen(result[len(result)-1])+1+runeLen(chunk) <= minSize*4 {
			result[len(result)-1] = strings.TrimSpace(result[len(result)-1] + "\n" + chunk)
			continue
		}
		// 否则保持为独立 chunk。
		result = append(result, chunk)
	}
	// 返回清洗后的 chunk 列表。
	return result
}

func overlapTail(text string, overlap, minSize int) string {
	// 从上一个 chunk 尾部取 overlap 大小的上下文，拼到下一个 chunk 头部。
	// 这样可以缓解“关键句正好卡在 chunk 边界”导致的召回损失。
	// 先把文本清洗后转成 rune，便于按字符截取。
	runes := []rune(strings.TrimSpace(text))
	// 没有内容或 overlap 非法时，不返回尾巴。
	if len(runes) == 0 || overlap <= 0 {
		return ""
	}
	// 如果 overlap 比文本本身还长，就截成文本长度。
	if overlap > len(runes) {
		overlap = len(runes)
	}
	// 取出尾部 overlap 长度的文本作为重叠上下文。
	tail := strings.TrimSpace(string(runes[len(runes)-overlap:]))
	// 如果尾巴太短，说明上下文价值有限，宁可不要。
	if runeLen(tail) < minSize/2 {
		return ""
	}
	// 返回可用的重叠尾巴。
	return tail
}

func normalizeContent(content string) string {
	// 统一换行风格，避免 Windows / Unix 文本在切分时行为不一致。
	// 先把 CRLF 转成 LF。
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// 再把单独的 CR 也转成 LF。
	content = strings.ReplaceAll(content, "\r", "\n")
	// 返回去掉首尾空白后的文本。
	return strings.TrimSpace(content)
}

func joinChunkParts(left, right string) string {
	// 这里用双换行拼接，是为了尽量保留原始段落边界。
	// 先清理左半部分首尾空白。
	left = strings.TrimSpace(left)
	// 再清理右半部分首尾空白。
	right = strings.TrimSpace(right)
	// 根据左右内容是否为空，决定如何拼接。
	switch {
	// 左边为空时，直接返回右边。
	case left == "":
		return right
	// 右边为空时，直接返回左边。
	case right == "":
		return left
	// 左右都非空时，用双换行拼起来保留段落边界。
	default:
		return left + "\n\n" + right
	}
}

func runeLen(text string) int {
	// 用 rune 数而不是字节数，避免中文文档按字节切分出现长度失真。
	// Go 里 string 的 len 默认返回字节数，这里显式转 rune。
	return len([]rune(text))
}

func isCodeFence(line string) bool {
	// 支持 ``` 和 ~~~ 两种 fenced code block 语法。
	return strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
}

func cloneMeta(src map[string]any) map[string]any {
	// 给每个 chunk 复制一份独立 metadata，避免多个 chunk 共享同一个 map 被后续覆盖。
	// 空 map 或 nil 时，直接返回一个新的空 map。
	if len(src) == 0 {
		return map[string]any{}
	}
	// 按原 map 容量创建新 map，减少扩容。
	dst := make(map[string]any, len(src))
	// 把原 metadata 的每个键值复制过去。
	for k, v := range src {
		dst[k] = v
	}
	// 返回复制后的 metadata。
	return dst
}
