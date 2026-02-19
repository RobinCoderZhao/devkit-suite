// Package prompt provides LLM prompt templates for DevKit CLI commands.
package prompt

// CommitPrompt is used to generate a conventional commit message from a diff.
const CommitPrompt = `你是一位高级软件工程师。根据下面的 git diff，生成一条符合 Conventional Commits 规范的 commit message。

规则：
1. 格式：<type>(<scope>): <description>
2. type 必须是以下之一：feat、fix、docs、style、refactor、perf、test、build、ci、chore
3. scope 是可选的，表示影响范围（如文件名、模块名）
4. description 用英文，简洁明了，不超过 72 个字符
5. 如果变更较大，可以在第二行空行后加详细说明（body），用中文
6. 不要加任何 markdown 格式，只输出纯文本 commit message

Git diff:
%s

请直接输出 commit message，不要加任何解释。`

// ReviewPrompt is used to generate a code review from a diff.
const ReviewPrompt = `你是一位资深代码审查员（Senior Code Reviewer）。请对以下 git diff 进行代码审查。

审查维度：
1. **Bug 风险**：是否有潜在的 bug、空指针、边界条件问题？
2. **安全性**：是否有安全漏洞（硬编码密钥、SQL 注入、XSS 等）？
3. **性能**：是否有性能问题（N+1 查询、不必要的内存分配等）？
4. **可读性**：代码是否清晰、命名是否合理？
5. **最佳实践**：是否遵循了语言和框架的最佳实践？
6. **测试**：变更是否应该有对应的测试？

输出 JSON 格式：
{
  "score": 8,          // 1-10 评分，10 为完美
  "summary": "总体评价一句话",
  "issues": [
    {
      "severity": "high|medium|low",
      "file": "文件名",
      "line": "行号或范围",
      "description": "问题描述",
      "suggestion": "建议修改"
    }
  ],
  "highlights": ["做得好的地方1", "做得好的地方2"]
}

Git diff:
%s`
