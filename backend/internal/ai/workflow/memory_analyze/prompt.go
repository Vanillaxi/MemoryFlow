package memory_analyze

import "fmt"

func BuildPrompt(input AnalyzeInput) string {
	switch input.Type {
	case TypeImage:
		return buildImagePrompt(input)
	case TypeMixed:
		return buildMixedPrompt(input)
	default:
		return buildTextPrompt(input)
	}
}

func buildTextPrompt(input AnalyzeInput) string {
	return fmt.Sprintf(`你是 MemoryFlow 的个人记忆分析助手。

请根据用户的一条生活记忆，提取结构化信息。

你必须严格输出 JSON 对象本身。
禁止输出 Markdown。
禁止使用代码块。
禁止输出解释文字。

字段要求：
1. summary：用一句中文总结这条记忆。
2. tags：提取 1 到 5 个中文标签。
3. mood：只能输出以下三个英文值之一：
   positive
   neutral
   negative
   禁止输出 accomplished、happy、sad、成就感、开心 等其他值。
4. importance_score：必须是 0 到 1 之间的小数。
   例如 0.1、0.5、0.8。
   禁止输出 1 以上的数字，例如 7、8、10。

输入：
内容：%s
地点：%s
时间：%s

请严格输出如下 JSON 格式：
{
  "summary": "一句话总结",
  "tags": ["标签1", "标签2"],
  "mood": "positive",
  "importance_score": 0.7
}
`, input.ContentText, input.Location, input.OccurredAt.Format("2006-01-02 15:04:05"))
}

func buildImagePrompt(input AnalyzeInput) string {
	return fmt.Sprintf("图片记忆：%s\n地点：%s\n时间：%s", input.ImageURL, input.Location, input.OccurredAt.Format("2006-01-02 15:04:05"))
}

func buildMixedPrompt(input AnalyzeInput) string {
	return fmt.Sprintf("图文记忆：%s\n说明：%s\n地点：%s\n时间：%s", input.ImageURL, input.ContentText, input.Location, input.OccurredAt.Format("2006-01-02 15:04:05"))
}
