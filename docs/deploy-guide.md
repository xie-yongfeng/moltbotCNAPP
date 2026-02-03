# 部署 GitHub Actions 工作流

## 当前状态

您已经创建了工作流文件，但还需要推送到 GitHub 才能激活。

## 📤 推送工作流到 GitHub

### 第一步：提交工作流文件

```bash
# 1. 查看新增的文件
git status

# 2. 添加工作流文件
git add .github/workflows/build.yml
git add docs/

# 3. 提交
git commit -m "ci: 添加 GitHub Actions 自动构建工作流"

# 4. 推送到 GitHub
git push origin main
```

### 第二步：验证工作流

推送后：
1. 访问 GitHub 仓库页面
2. 点击顶部 **Actions** 标签
3. **刷新页面**（重要！）
4. 现在您应该能看到：
   - 左侧出现 "Build and Release" 工作流
   - 右侧出现绿色的 **Run workflow** 按钮

### 第三步：手动触发构建

1. 点击左侧 **Build and Release** 工作流名称
2. 右侧会出现绿色的 **Run workflow** 按钮
3. 点击按钮，选择分支（main）
4. 点击绿色 **Run workflow** 确认

## 🎯 完整命令（一键执行）

```bash
# 在项目根目录执行
cd /Users/xcyy/work/moltbotCNAPP

# 提交并推送
git add .
git commit -m "ci: 添加 GitHub Actions 自动构建和 session_key 配置"
git push origin main

# 等待几秒后刷新 GitHub Actions 页面
```

## 📸 预期效果

推送后刷新 Actions 页面，您应该看到：

```
Actions
├── All workflows
└── Build and Release  [右侧有 "Run workflow" 按钮]
    └── Runs (运行记录将显示在这里)
```

## ⚠️ 故障排查

### 看不到 Run workflow 按钮？

**原因：** 工作流文件还未推送或页面未刷新

**解决方法：**
1. 确认文件已推送：`git log --oneline -1`
2. 强制刷新浏览器：Ctrl+Shift+R (Windows) 或 Cmd+Shift+R (Mac)
3. 等待 1-2 分钟让 GitHub 处理

### 工作流不出现？

**检查文件位置：**
```bash
ls -la .github/workflows/build.yml
```

文件必须在正确的路径：`.github/workflows/build.yml`

## 🚀 创建首次构建

### 方式一：手动触发（推荐新手）

按照上面的步骤点击 "Run workflow" 按钮

### 方式二：创建 Tag（推荐发布）

```bash
# 创建并推送标签
git tag v0.1.0
git push origin v0.1.0
```

这会自动触发构建并创建 Release

## ✅ 验证成功

构建成功后，您可以在以下位置找到 Windows exe：

1. **Artifacts**（手动触发）：
   - Actions → 选择运行记录 → Artifacts 区域

2. **Releases**（Tag 触发）：
   - 仓库首页 → 右侧 Releases → 选择版本 → Assets

## 💡 提示

- 首次推送工作流文件后，可能需要等待 1-2 分钟
- 刷新浏览器（Ctrl+F5）确保看到最新状态
- 如果仍然看不到，检查仓库设置中是否启用了 Actions