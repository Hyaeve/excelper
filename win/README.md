# Excelper Windows PC 版

这个目录用于生成和放置 Windows 桌面版启动包。

## 产物结构

构建完成后，目录结构为：

```text
win/
  package/
    Excelper.exe
    frontend-dist/
    data/
```

双击 `package/Excelper.exe` 后会启动本地服务，并自动打开：

```text
http://127.0.0.1:3012
```

## 使用方式

1. 把要处理的 `.xls` 文件放入 `win/package/data`。
2. 双击 `win/package/Excelper.exe`。
3. 在页面中选择文件、填写规则、预览并执行。

## 本地启动脚本

如果还没有生成 `Excelper.exe`，可以先在项目根目录运行：

```bat
win\start.bat
```

或运行 PowerShell 脚本：

```powershell
.\win\start.ps1
```

启动脚本会优先运行 `win/package/Excelper.exe`；如果 exe 不存在，会尝试用本机 Go 临时启动后端，并在缺少 `frontend/dist` 时用 npm 构建前端，然后打开本地浏览器。

## 构建方式

在项目根目录运行 PowerShell 构建脚本：

```powershell
.\win\build.ps1
```

也可以使用批处理脚本：

```bat
win\build.bat
```

构建脚本会：

- 构建 Vue 前端到 `frontend/dist`
- 编译 Windows 版 `Excelper.exe`
- 复制前端静态资源到 `win/package/frontend-dist`
- 创建 `win/package/data` 数据目录

## 运行依赖

当前 `.xls` 实际写回仍依赖 LibreOffice 命令行转换能力。PC 版运行机器需要安装 LibreOffice，并确保 `soffice.exe` 可从系统 PATH 调用。也可以通过 `EXCELPER_OFFICE_COMMAND` 指定完整命令路径。

可以用环境变量自定义目录和端口：

```bat
set EXCELPER_DATA_DIR=D:\ExcelperData
set EXCELPER_PORT=3012
Excelper.exe
```

## 图标

简洁线条图标源文件位于 `win/assets/excelper-icon.svg`。如果后续需要嵌入到 exe，可用 Windows 资源工具将该图标转换为 `.ico` 后写入可执行文件资源。