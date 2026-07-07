# Excelper

`Excelper` 是一个基于 Docker 的 [`Go`](go.mod) + [`Vue 3`](frontend/package.json) 项目，用于对 Docker 挂载目录中的单个 [`xls`](README.md) Excel 文件进行预览、规则录入、执行输出与自动备份。

## 功能特性

- 仅处理 Docker 挂载目录 [`/data`](docker-compose.yml) 中的单个 `.xls` 文件
- 左侧显示原始表格预览
- 中间填写录入规则并点击“预览”或“执行”
- 右侧显示录入结果预览
- 每次执行时自动备份原始文件
- 备份文件与输出文件都按“年月日时分”时间戳命名

## 规则语法

界面中需要填写以下内容：

- 目标列：例如 `B`
- 起始行：例如 `924`，每次录入语法操作都必须填写
- 固定前缀：例如 `25B140-`
- 固定后缀：例如 `自交`
- 中间变化值：使用中文顿号 `、` 分隔

示例规则：

`51、in58、66、in71、73、76`

说明：

- `51` 表示直接写入当前目标行
- `in58` 表示该值对应的是一个原表不存在的新行，系统会把它标记为插入项
- 每项最终会拼接成：`固定前缀 + 值 + 固定后缀`
- 例如：`25B140-51自交`
- `in` 大小写不敏感，`in58` 和 `IN58` 都会按插入项处理

## 当前预览与执行逻辑

当前版本已实现：

- 枚举挂载目录中的 `.xls` 文件
- 读取并预览原始表格内容
- 根据规则生成右侧录入预览
- 执行时生成：
  - 备份文件：`原文件名_backup_时间戳.xls`
  - 输出文件：`原文件名_output_时间戳.xls`

当前后端中的 [`executeRule()`](main.go:109) 已完成文件备份与输出文件生成；规则预览由 [`previewRule()`](main.go:81) 与 [`buildGeneratedPreview()`](main.go:191) 提供。

## 界面结构

前端页面位于 [`frontend/src/App.vue`](frontend/src/App.vue)，布局遵循你提供的示意图：

- 左栏：原始文件预览
- 中栏：规则录入区 + 预览/执行按钮
- 右栏：录入结果预览

样式位于 [`frontend/src/style.css`](frontend/src/style.css)，采用浅色磨砂卡片风格。

## 项目结构

- 后端入口：[`main.go`](main.go)
- 前端入口：[`frontend/src/main.js`](frontend/src/main.js)
- 前端页面：[`frontend/src/App.vue`](frontend/src/App.vue)
- Docker 构建：[`Dockerfile`](Dockerfile)
- 容器编排：[`docker-compose.yml`](docker-compose.yml)

## Docker 使用方式

### 1. 准备挂载目录

在项目根目录下准备一个用于挂载的目录，例如：

- [`mounted-data`](mounted-data)

把需要处理的 `.xls` 文件放入该目录。

### 2. 启动项目

当前 [`docker-compose.yml`](docker-compose.yml) 使用已发布镜像：

```yaml
image: ghcr.io/hyaeve/excelper:latest
```

启动命令：

```bash
docker compose up -d
```

启动后访问：

- `http://localhost:3012`

### 3. 使用流程

1. 在左侧选择挂载目录中的 `.xls` 文件
2. 预览原始表格
3. 在中间输入目标列、起始行、固定前缀、固定后缀、规则串
4. 确认起始行已填写，例如 `924`
5. 点击“预览”查看右侧生成结果
6. 点击“执行”后生成备份文件和输出文件

## 镜像发布

项目包含 GitHub Actions 工作流 [`docker-publish.yml`](.github/workflows/docker-publish.yml)。当代码推送到 `main` 分支时，会自动构建 [`Dockerfile`](Dockerfile)，并推送镜像到：

```text
ghcr.io/hyaeve/excelper:latest
```

[`docker-compose.yml`](docker-compose.yml) 会直接拉取该镜像运行。

## 挂载目录说明

当前 [`docker-compose.yml`](docker-compose.yml) 中将本地目录挂载到容器内目录 [`/data`](docker-compose.yml)：

```yaml
volumes:
  - ./mounted-data:/data
```

程序只读取容器内的 [`/data`](docker-compose.yml) 目录，不直接浏览宿主机任意路径，这符合“只能添加挂载文件目录中的文件”的要求。

## 后续可继续增强的点

当前版本已完成界面、预览、备份与输出流程骨架。若继续迭代，建议增强：

- 在 [`executeRule()`](main.go:109) 中真正写入 `.xls` 单元格内容
- 根据 `in` 规则对目标工作表执行真实插行
- 支持选择具体工作表
- 右侧结果区增加“下载输出文件”按钮
- 左右两栏增加更接近 Excel 的网格高亮与定位能力
