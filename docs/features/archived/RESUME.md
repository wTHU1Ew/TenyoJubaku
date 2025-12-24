# 快速恢复命令 / Quick Resume Commands

如果 Claude Code 会话中断或 token 用完，使用以下命令快速恢复工作。

---

## 🚀 推荐命令 / Recommended Commands

### 继续 Phase 1B 重构
```
继续 PHASE1B_TODO.md 中的工作
```

### 查看当前进度
```
查看 PHASE1B_TODO.md 的完成进度
```

### 从特定阶段开始
```
从 Phase 1B.2 (Storage Context Methods) 开始
```

---

## 📝 其他有用命令 / Other Useful Commands

### 查看架构状态
```
总结 PHASE1A_STATUS.md 的内容
```

### 查看实施指南
```
使用 REFACTORING_GUIDE.md 实现 OKX Client Context 方法
```

### 验证代码状态
```
验证当前代码能否编译，Feature 1 和 2 是否正常工作
```

---

## 🎯 快速上下文恢复 / Quick Context Recovery

如果需要快速了解项目状态，告诉 Claude：

```
读取以下文件了解当前状态：
1. PHASE1A_STATUS.md - 当前状态
2. PHASE1B_TODO.md - 待办任务
3. REFACTORING_GUIDE.md - 实施指南
```

---

## 📊 当前项目状态快照 / Current Project Status Snapshot

**日期**: 2025-12-03
**Phase**: 1B 待开始 / 1B Not Started
**Feature 1 & 2**: ✅ 正常工作 / Working
**Feature 3**: ⏳ 基础就绪，等待实施 / Foundation Ready

**总任务数 / Total Tasks**: 47
**已完成 / Completed**: 0
**进度 / Progress**: 0%

---

## 💡 提示 / Tips

1. **简短命令最有效**: "继续 PHASE1B_TODO.md" 比长篇解释更好
2. **引用文件名**: 直接说文件名，我会读取并理解上下文
3. **指定阶段**: 如果只想做特定部分，明确说明（如 "只做 Phase 1B.1"）
4. **验证优先**: 每次恢复后先验证代码状态

---

## 🔧 调试命令 / Debug Commands

### 检查编译状态
```
运行 go build ./... 检查是否有编译错误
```

### 检查接口实现
```
检查 okx.Client 和 storage.Storage 是否实现了对应的接口
```

### 查看未完成任务
```
列出 PHASE1B_TODO.md 中所有未完成的任务
```
