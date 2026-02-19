// Package analyzer uses LLM to analyze, deduplicate, and summarize AI news articles.
package analyzer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RobinCoderZhao/API-Change-Sentinel/internal/newsbot/sources"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
)

// DailyDigest is the final output of the analyzer.
type DailyDigest struct {
	Date        string     `json:"date"`
	Headlines   []Headline `json:"headlines"`
	Summary     string     `json:"summary"`
	GeneratedAt time.Time  `json:"generated_at"`
	TokensUsed  int        `json:"tokens_used"`
	Cost        float64    `json:"cost"`
}

// Headline is a single news item in the digest.
type Headline struct {
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	URL        string   `json:"url"`
	Source     string   `json:"source"`
	Importance string   `json:"importance"` // "high", "medium", "low"
	Tags       []string `json:"tags"`
}

// Analyzer processes raw articles into a curated daily digest.
type Analyzer struct {
	client llm.Client
}

// NewAnalyzer creates a new article analyzer with the given LLM client.
func NewAnalyzer(client llm.Client) *Analyzer {
	return &Analyzer{client: client}
}

// Analyze takes raw articles and produces a DailyDigest.
func (a *Analyzer) Analyze(ctx context.Context, articles []sources.Article) (*DailyDigest, error) {
	if len(articles) == 0 {
		return &DailyDigest{
			Date:        time.Now().Format("2006-01-02"),
			GeneratedAt: time.Now(),
		}, nil
	}

	// Build article summaries for LLM input
	var sb strings.Builder
	for i, art := range articles {
		if i >= 50 { // Limit to 50 articles to stay within context window
			break
		}
		content := art.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("---\n[%d] Title: %s\nSource: %s\nURL: %s\nContent: %s\n",
			i+1, art.Title, art.Source, art.URL, content))
	}

	resp, err := a.client.Generate(ctx, &llm.Request{
		System: analyzerSystemPrompt,
		Messages: []llm.Message{
			{Role: "user", Content: fmt.Sprintf("今天是 %s。\n\n以下是今天收集到的 AI 相关新闻：\n\n%s", time.Now().Format("2006-01-02"), sb.String())},
		},
		JSONMode: true,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	var digest DailyDigest
	if err := a.client.GenerateJSON(ctx, &llm.Request{
		System: analyzerSystemPrompt,
		Messages: []llm.Message{
			{Role: "user", Content: fmt.Sprintf("今天是 %s。\n\n以下是今天收集到的 AI 相关新闻：\n\n%s", time.Now().Format("2006-01-02"), sb.String())},
		},
	}, &digest); err != nil {
		// Fallback: use the raw text result
		digest = DailyDigest{
			Date:    time.Now().Format("2006-01-02"),
			Summary: resp.Content,
		}
	}

	digest.Date = time.Now().Format("2006-01-02")
	digest.GeneratedAt = time.Now()
	digest.TokensUsed = resp.TokensIn + resp.TokensOut
	digest.Cost = resp.Cost

	return &digest, nil
}

const analyzerSystemPrompt = `你是一位 AI 领域的资深编辑，负责制作每日 AI 热点日报。

你的任务：
1. 从给定的新闻列表中，筛选出最重要的 5-10 条 AI 相关新闻
2. 去重：相同事件的多篇报道合并为一条
3. 按重要性排序（high > medium > low）
4. 为每条新闻写一句话摘要（中文，30 字以内）
5. 生成一段总结（3-5 句话，概括今日 AI 领域大事）

重要性判断标准：
- HIGH：大公司发布新模型、重大融资、行业政策变化、技术突破
- MEDIUM：新产品发布、开源项目、研究论文
- LOW：评论文章、教程、小更新

输出 JSON 格式：
{
  "headlines": [
    {
      "title": "原标题",
      "summary": "一句话中文摘要",
      "url": "原文链接",
      "source": "来源",
      "importance": "high/medium/low",
      "tags": ["标签1", "标签2"]
    }
  ],
  "summary": "今日 AI 领域总结（中文）"
}`
