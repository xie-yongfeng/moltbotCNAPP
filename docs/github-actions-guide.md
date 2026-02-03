# GitHub Actions 自动构建指南

## 📋 概述

项目已配置 GitHub Actions 自动构建工作流，可以自动编译多平台二进制文件（包括 Windows exe）。

## 🚀 触发方式

### 方式一：创建 Git Tag（推荐用于正式发布）

```bash
# 1. 提交所有更改
git add .
git commit -m "feat: 添加 session_key 配置字段"

# 2. 创建版本标签
git tag v0.1.0

# 3. 推送代码和标签到 GitHub
git push origin main
git push origin v0.1.0
```

**自动执行：**
- ✅ 编译所有平台二进制文件
- ✅ 创建 GitHub Release
- ✅ 自动上传所有构建文件到 Release

### 方式二：手动触发

1. 访问 GitHub 仓库页面
2. 点击 **Actions** 标签
3. 选择 **Build and Release** 工作流
4. 点击右侧 **Run workflow** 按钮
5. 选择分支，点击 **Run workflow**

**自动执行：**
- ✅ 编译所有平台二进制文件
- ✅ 上传为工作流 Artifacts（保留 90 天）
- ❌ 不创建 Release（仅用于测试）

## 📦 生成的文件

工作流会生成以下平台的二进制文件：

| 文件名 | 平台 | 架构 |
|--------|------|------|
| `clawdbot-bridge-linux-amd64` | Linux | x86_64 |
| `clawdbot-bridge-linux-arm64` | Linux | ARM64 |
| `clawdbot-bridge-darwin-amd64` | macOS | Intel |
| `clawdbot-bridge-darwin-arm64` | macOS | Apple Silicon |
| `clawdbot-bridge-windows-amd64.exe` | Windows | x86_64 |
| `clawdbot-bridge-windows-arm64.exe` | Windows | ARM64 |

## 📥 下载构建文件

### 从 Release 下载（Tag 触发）

1. 访问仓库的 **Releases** 页面
2. 选择对应版本（如 v0.1.0）
3. 在 **Assets** 区域下载需要的文件

### 从 Artifacts 下载（手动触发）

1. 访问仓库的 **Actions** 标签
2. 选择对应的工作流运行记录
3. 在 **Artifacts** 区域下载 `clawdbot-bridge-{version}` 压缩包

## 🔧 配置文件说明

工作流配置文件位于：`.github/workflows/build.yml`

### 关键配置项

```yaml
on:
  push:
    tags:
      - 'v*'           # 当推送 v* 标签时触发
  workflow_dispatch:   # 允许手动触发
```

### 构建步骤

1. **Checkout code** - 检出代码
2. **Set up Go** - 安装 Go 1.21
3. **Get version** - 获取版本号
4. **Build all platforms** - 执行 `scripts/build.sh` 编译
5. **Upload artifacts** - 上传构建文件
6. **Create Release** - （仅 Tag 触发）创建 GitHub Release

## 📝 完整发布流程示例

```bash
# 1. 确保所有更改已提交
git status

# 2. 添加并提交更改
git add .
git commit -m "feat: 添加新功能"

# 3. 创建版本标签（遵循语义化版本）
git tag v0.1.0

# 4. 推送到 GitHub
git push origin main
git push origin v0.1.0

# 5. 等待几分钟，GitHub Actions 自动构建

# 6. 访问 Releases 页面下载编译好的文件
```

## 🎯 版本号规范

建议使用语义化版本号（Semantic Versioning）：

- `v0.1.0` - 初始版本
- `v0.1.1` - Bug 修复
- `v0.2.0` - 新增功能（向后兼容）
- `v1.0.0` - 重大更新（可能不向后兼容）

## 🔍 查看构建状态

### 实时监控

1. 推送 Tag 后，访问仓库的 **Actions** 标签
2. 可以看到正在运行的工作流
3. 点击进入查看详细日志

### 构建徽章

可以在 README 中添加构建状态徽章：

```markdown
![Build Status](https://github.com/wy51ai/moltbotCNAPP/actions/workflows/build.yml/badge.svg)
```

## ⚠️ 注意事项

1. **首次使用**：确保仓库设置中启用了 GitHub Actions
2. **Secrets**：无需额外配置，使用自动提供的 `GITHUB_TOKEN`
3. **权限**：确保仓库有创建 Release 的权限
4. **分支保护**：如果设置了分支保护，需要配置相应规则

## 🛠️ 本地测试

在推送到 GitHub 之前，可以本地测试构建：

```bash
# 测试构建脚本
chmod +x scripts/build.sh
./scripts/build.sh

# 检查生成的文件
ls -lh dist/
```

## 📞 故障排查

### 构建失败

1. 查看 Actions 日志定位错误
2. 常见问题：
   - Go 版本不兼容
   - 依赖包下载失败
   - 构建脚本权限问题

### Release 创建失败

1. 检查是否有同名 Tag 的 Release
2. 检查仓库权限设置
3. 查看工作流日志中的详细错误信息

## 🎉 快速开始

```bash
# 完整的发布流程（一键执行）
git add . && \
git commit -m "release: v0.1.0" && \
git tag v0.1.0 && \
git push origin main && \
git push origin v0.1.0

echo "✅ 已推送，请访问 GitHub Actions 查看构建进度"
echo "🔗 https://github.com/wy51ai/moltbotCNAPP/actions"