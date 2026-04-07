# NetDispatch 项目 AI 指南

> **AI 配置已迁移**：项目的完整 AI 能力配置现已整合到 `.claude/` 目录。

---

## 快速开始

### Claude Code 用户

项目已配置以下 skills：

| Skill | 说明 |
|-------|------|
| `project-context` | 项目上下文、开发指南、常见问题 |
| `release` | 发布流程、版本管理、GitHub Actions |

启动 Claude Code 后，这些 skills 会自动加载。

### 其他 AI 工具用户

阅读以下文件获取完整项目能力：

```bash
# 项目上下文和开发指南
.claude/skills/project-context.md

# 发布流程
.claude/skills/release.md
```

---

## 文件位置

| 内容 | 文件路径 |
|------|----------|
| 项目上下文 | `.claude/skills/project-context.md` |
| 发布流程 | `.claude/skills/release.md` |
| 详细架构 | `docs/architecture.md` |
| 编译指南 | `docs/build-guide.md` |
| 项目介绍 | `README.md` |

---

## 为其他 AI 工具配置

### Gemini CLI

创建 `GEMINI.md` 或在项目根目录添加配置，指向 `.claude/skills/` 中的文件。

### Cursor / 其他 IDE

在 `.cursorrules` 或类似配置中引用 `.claude/skills/project-context.md`。
