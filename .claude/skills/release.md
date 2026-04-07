---
name: release
description: When making significant code changes, create a release tag to trigger GitHub Actions automatic build and publish
---

# Release Workflow

## 自动触发条件

**重要**：当完成以下类型的重大代码变动时，Agent 应主动建议或执行发布流程：

| 变动类型 | 说明 | 版本增量 |
|---------|------|---------|
| 新功能 | 添加重要的新功能模块 | MINOR |
| 重大重构 | 架构调整、模块重写 | MINOR |
| 关键修复 | 修复严重 bug 或安全问题 | PATCH |
| API 变更 | 接口不兼容的修改 | MAJOR |
| 工作流新增 | 添加 CI/CD、自动化流程 | MINOR |

### 判断标准

以下情况应考虑触发发布：

1. **功能完整性** - 完成了独立可用的功能模块
2. **代码质量** - 通过测试，本地构建成功
3. **用户价值** - 对用户有实际价值，值得分发
4. **文档完备** - 相关文档已更新

## 发布流程

### 1. 确认发布版本号

查看当前最新版本：

```bash
git tag --sort=-v:refname | head -5
```

根据变更类型确定新版本号：

```bash
# 当前最新版本: v1.0.0
# 新功能/重构 → v1.1.0
# Bug 修复 → v1.0.1
# 不兼容变更 → v2.0.0
```

### 2. 本地验证

```bash
# 构建前端
cd web && npm install && npm run build && cd ..

# 构建并测试
go build -o netdispatch.exe ./cmd/netdispatch
./netdispatch.exe start
```

### 3. 提交代码（如有未提交的变更）

```bash
git add .
git commit -m "feat: 添加 GitHub Actions 自动发布功能"
```

### 4. 创建并推送 Tag

```bash
# 创建 tag
git tag v1.1.0

# 推送代码和 tag
git push origin main
git push origin v1.1.0
```

### 5. 等待 GitHub Actions 完成

推送 tag 后，GitHub Actions 自动执行：

1. 构建前端
2. 生成 Changelog（从 commits）
3. 构建二进制文件：
   - `netdispatch-windows-amd64.exe`
   - `netdispatch-linux-amd64`
4. 创建 GitHub Release

### 6. 验证发布

访问 GitHub Repository → Releases 确认发布成功。

## 版本命名规范

使用语义化版本：`vMAJOR.MINOR.PATCH`

- **MAJOR** - 不兼容的 API 变更
- **MINOR** - 向后兼容的新功能
- **PATCH** - Bug 修复

## Agent 行为指南

当判断需要发布时，Agent 应：

1. **主动提示** - 告知用户即将创建发布版本
2. **确认版本号** - 与用户确认版本号是否正确
3. **执行发布** - 创建 tag 并推送

示例对话：

```
Agent: 已完成 GitHub Actions 自动发布功能的实现。这是一个重大功能变更，建议发布新版本。

当前版本: v1.0.0
建议版本: v1.1.0

是否创建发布？
```

## 工作流配置

发布工作流定义在 `.github/workflows/release.yml`。

## 默认配置嵌入

二进制包含嵌入的默认配置 (`pkg/config/default.yaml`)。首次启动时自动创建配置文件：

| 平台 | 配置路径 |
|------|----------|
| Windows | `%APPDATA%/NetDispatch/config.yaml` |
| Linux/macOS | `~/.config/NetDispatch/config.yaml` |
