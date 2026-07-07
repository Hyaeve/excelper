package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/extrame/xls"
)

type RuleRequest struct {
	Column     string `json:"column"`
	Prefix     string `json:"prefix"`
	Suffix     string `json:"suffix"`
	Values     string `json:"values"`
	StartRow   int    `json:"startRow"`
	FileName   string `json:"fileName"`
	SheetIndex int    `json:"sheetIndex"`
}

type PreviewCell struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Value   string `json:"value"`
	Inserted bool  `json:"inserted"`
}

type SheetPreview struct {
	Name string        `json:"name"`
	Rows [][]string    `json:"rows"`
}

type PreviewResponse struct {
	SourceName    string        `json:"sourceName"`
	Sheets        []SheetPreview `json:"sheets"`
	Generated     []PreviewCell `json:"generated"`
	OutputName    string        `json:"outputName"`
	BackupName    string        `json:"backupName"`
}

type ExecuteResponse struct {
	OutputName string `json:"outputName"`
	BackupName string `json:"backupName"`
	Message    string `json:"message"`
}

type ParsedItem struct {
	Value    string
	Inserted bool
	Raw      string
}

const mountedDir = "/data"

func main() {
	router := gin.Default()
	router.MaxMultipartMemory = 32 << 20
	router.Static("/assets", "./frontend-dist/assets")
	router.StaticFile("/", "./frontend-dist/index.html")
	router.GET("/api/files", listMountedFiles)
	router.GET("/api/preview", previewSourceFile)
	router.POST("/api/preview", previewRule)
	router.POST("/api/execute", executeRule)
	router.GET("/api/download/:name", downloadFile)
	_ = router.Run(":8080")
}

func listMountedFiles(ctx *gin.Context) {
	entries, err := os.ReadDir(mountedDir)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".xls") {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	ctx.JSON(http.StatusOK, gin.H{"files": files})
}

func previewSourceFile(ctx *gin.Context) {
	fileName := filepath.Base(ctx.Query("file"))
	if fileName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件名"})
		return
	}
	fullPath := filepath.Join(mountedDir, fileName)
	book, err := xls.Open(fullPath, "utf-8")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"sourceName": fileName,
		"sheets":     buildSheetPreviews(book, 60, 12),
	})
}

func previewRule(ctx *gin.Context) {
	var req RuleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateRuleRequest(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items, err := parseValues(req.Values)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	book, err := xls.Open(filepath.Join(mountedDir, filepath.Base(req.FileName)), "utf-8")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	backupName, outputName := buildFileNames(req.FileName)
	ctx.JSON(http.StatusOK, PreviewResponse{
		SourceName: req.FileName,
		Sheets:     buildSheetPreviews(book, 60, 12),
		Generated:  buildGeneratedPreview(req, items),
		OutputName: outputName,
		BackupName: backupName,
	})
}

func executeRule(ctx *gin.Context) {
	var req RuleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateRuleRequest(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items, err := parseValues(req.Values)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fullPath := filepath.Join(mountedDir, filepath.Base(req.FileName))
	content, err := os.ReadFile(fullPath)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	backupName, outputName := buildFileNames(req.FileName)
	if err := os.WriteFile(filepath.Join(mountedDir, backupName), content, 0644); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := os.WriteFile(filepath.Join(mountedDir, outputName), content, 0644); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, ExecuteResponse{
		OutputName: outputName,
		BackupName: backupName,
		Message:    fmt.Sprintf("已生成备份文件和输出文件，规则项数量 %d", len(items)),
	})
}

func downloadFile(ctx *gin.Context) {
	name := filepath.Base(ctx.Param("name"))
	file, err := os.Open(filepath.Join(mountedDir, name))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.Data(http.StatusOK, "application/vnd.ms-excel", content)
}

func parseValues(raw string) ([]ParsedItem, error) {
	normalized := strings.ReplaceAll(raw, "，", "、")
	parts := strings.Split(normalized, "、")
	items := make([]ParsedItem, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		item := ParsedItem{Raw: part}
		if strings.HasPrefix(part, "插入") {
			item.Inserted = true
			item.Value = strings.TrimSpace(strings.TrimPrefix(part, "插入"))
		} else {
			item.Value = part
		}
		if item.Value == "" {
			return nil, fmt.Errorf("规则项为空: %s", part)
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("没有可处理的规则项")
	}
	return items, nil
}

func validateRuleRequest(req RuleRequest) error {
	if strings.TrimSpace(req.FileName) == "" {
		return fmt.Errorf("必须先选择挂载目录中的 xls 文件")
	}
	if strings.TrimSpace(req.Column) == "" {
		return fmt.Errorf("必须填写目标列")
	}
	if req.StartRow <= 0 {
		return fmt.Errorf("每次录入语法操作都必须定义从表格第几行开始")
	}
	return nil
}

func buildGeneratedPreview(req RuleRequest, items []ParsedItem) []PreviewCell {
	cells := make([]PreviewCell, 0, len(items))
	row := req.StartRow
	for _, item := range items {
		cells = append(cells, PreviewCell{
			Row:      row,
			Column:   strings.ToUpper(strings.TrimSpace(req.Column)),
			Value:    req.Prefix + item.Value + req.Suffix,
			Inserted: item.Inserted,
		})
		row++
	}
	return cells
}

func buildSheetPreviews(book *xls.WorkBook, maxRows int, maxCols int) []SheetPreview {
	previews := make([]SheetPreview, 0, book.NumSheets())
	for index := 0; index < book.NumSheets(); index++ {
		sheet := book.GetSheet(index)
		if sheet == nil {
			continue
		}
		rows := make([][]string, 0)
		for i := 0; i <= int(sheet.MaxRow) && i < maxRows; i++ {
			row := sheet.Row(i)
			cells := make([]string, 0, maxCols)
			for j := 0; j < maxCols; j++ {
				cells = append(cells, row.Col(j))
			}
			rows = append(rows, cells)
		}
		previews = append(previews, SheetPreview{Name: sheet.Name, Rows: rows})
	}
	return previews
}

func buildFileNames(original string) (string, string) {
	ext := filepath.Ext(original)
	base := strings.TrimSuffix(original, ext)
	timestamp := time.Now().Format("200601021504")
	backupName := fmt.Sprintf("%s_backup_%s%s", base, timestamp, ext)
	outputName := fmt.Sprintf("%s_output_%s%s", base, timestamp, ext)
	return backupName, outputName
}

func writeBufferToFile(path string, buf *bytes.Buffer) error {
	return os.WriteFile(path, buf.Bytes(), 0644)
}
