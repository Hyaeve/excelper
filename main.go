package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/extrame/xls"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
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
	Name      string     `json:"name"`
	Rows      [][]string `json:"rows"`
	RowOffset int        `json:"rowOffset"`
	HasMore   bool       `json:"hasMore"`
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
	_ = router.Run(":3012")
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
	rowOffset := parseQueryInt(ctx, "offset", 0)
	rowLimit := parseQueryInt(ctx, "limit", 60)
	fullPath := filepath.Join(mountedDir, fileName)
	book, err := xls.Open(fullPath, "utf-8")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"sourceName": fileName,
		"sheets":     buildSheetPreviews(book, rowOffset, rowLimit, 12),
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
	backupName := buildBackupName(req.FileName)
	outputName := filepath.Base(req.FileName)
	ctx.JSON(http.StatusOK, PreviewResponse{
		SourceName: req.FileName,
		Sheets:     buildSheetPreviews(book, 0, 60, 12),
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
	backupName := buildBackupName(req.FileName)
	backupPath := filepath.Join(mountedDir, backupName)
	if err := copyFile(fullPath, backupPath); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := applyRuleToXLS(fullPath, req, items); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	outputName := filepath.Base(req.FileName)
	ctx.JSON(http.StatusOK, ExecuteResponse{
		OutputName: outputName,
		BackupName: backupName,
		Message:    fmt.Sprintf("已备份原文件并覆盖写入，规则项数量 %d", len(items)),
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
	normalized := strings.NewReplacer("，", " ", "、", " ", ",", " ").Replace(raw)
	parts := strings.Fields(normalized)
	items := make([]ParsedItem, 0, len(parts))
	for _, part := range parts {
		expanded, err := expandRulePart(part)
		if err != nil {
			return nil, err
		}
		items = append(items, expanded...)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("没有可处理的规则项")
	}
	return items, nil
}

func expandRulePart(part string) ([]ParsedItem, error) {
	inserted := strings.HasPrefix(part, "+")
	value := part
	if inserted {
		value = strings.TrimSpace(strings.TrimPrefix(part, "+"))
	}
	if value == "" {
		return nil, fmt.Errorf("规则项为空: %s", part)
	}
	if strings.Count(value, "-") == 1 {
		bounds := strings.SplitN(value, "-", 2)
		start, startErr := strconv.Atoi(bounds[0])
		end, endErr := strconv.Atoi(bounds[1])
		if startErr == nil && endErr == nil {
			if end < start {
				return nil, fmt.Errorf("区间规则结束值不能小于开始值: %s", part)
			}
			items := make([]ParsedItem, 0, end-start+1)
			for number := start; number <= end; number++ {
				items = append(items, ParsedItem{
					Raw:      part,
					Value:    strconv.Itoa(number),
					Inserted: inserted,
				})
			}
			return items, nil
		}
	}
	return []ParsedItem{{Raw: part, Value: value, Inserted: inserted}}, nil
}

func validateRuleRequest(req RuleRequest) error {
	if strings.TrimSpace(req.FileName) == "" {
		return fmt.Errorf("必须先选择挂载目录中的 xls 文件")
	}
	if _, err := columnNameToNumber(req.Column); err != nil {
		return err
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
			Value:    composeCellValue(req, item.Value),
			Inserted: item.Inserted,
		})
		row++
	}
	return cells
}

func parseQueryInt(ctx *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(ctx.DefaultQuery(key, strconv.Itoa(fallback)))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func buildSheetPreviews(book *xls.WorkBook, rowOffset int, rowLimit int, maxCols int) []SheetPreview {
	if rowLimit <= 0 || rowLimit > 200 {
		rowLimit = 60
	}
	previews := make([]SheetPreview, 0, book.NumSheets())
	for index := 0; index < book.NumSheets(); index++ {
		sheet := book.GetSheet(index)
		if sheet == nil {
			continue
		}
		maxRow := int(sheet.MaxRow)
		rows := make([][]string, 0, rowLimit)
		endRow := rowOffset + rowLimit
		if endRow > maxRow+1 {
			endRow = maxRow + 1
		}
		for i := rowOffset; i < endRow; i++ {
			row := sheet.Row(i)
			cells := make([]string, 0, maxCols)
			for j := 0; j < maxCols; j++ {
				cells = append(cells, row.Col(j))
			}
			rows = append(rows, cells)
		}
		previews = append(previews, SheetPreview{
			Name:      sheet.Name,
			Rows:      rows,
			RowOffset: rowOffset,
			HasMore:   endRow <= maxRow,
		})
	}
	return previews
}

func buildBackupName(original string) string {
	ext := filepath.Ext(original)
	base := strings.TrimSuffix(filepath.Base(original), ext)
	timestamp := time.Now().Format("200601021504")
	return fmt.Sprintf("%s_%s%s", base, timestamp, ext)
}

func columnNameToNumber(column string) (int, error) {
	column = strings.ToUpper(strings.TrimSpace(column))
	if column == "" {
		return 0, fmt.Errorf("必须填写目标列")
	}
	columnNumber := 0
	for _, char := range column {
		if char < 'A' || char > 'Z' {
			return 0, fmt.Errorf("目标列格式错误: %s", column)
		}
		columnNumber = columnNumber*26 + int(char-'A'+1)
	}
	return columnNumber, nil
}

func applyRuleToXLS(path string, req RuleRequest, items []ParsedItem) error {
	columnNumber, err := columnNameToNumber(req.Column)
	if err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp("", "excelper-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := convertOfficeFile(path, "xlsx", tempDir); err != nil {
		return err
	}
	xlsxPath := filepath.Join(tempDir, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))+".xlsx")
	workbook, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		return err
	}
	defer workbook.Close()

	sheetName, err := sheetNameByIndex(workbook, req.SheetIndex)
	if err != nil {
		return err
	}
	rowNumber := req.StartRow
	for _, item := range items {
		if item.Inserted {
			rowHeight, err := workbook.GetRowHeight(sheetName, rowNumber)
			if err != nil {
				return err
			}
			if err := workbook.InsertRows(sheetName, rowNumber, 1); err != nil {
				return err
			}
			if err := workbook.SetRowHeight(sheetName, rowNumber, rowHeight); err != nil {
				return err
			}
		}
		cellName, err := excelize.CoordinatesToCellName(columnNumber, rowNumber)
		if err != nil {
			return err
		}
		if err := workbook.SetCellStr(sheetName, cellName, composeCellValue(req, item.Value)); err != nil {
			return err
		}
		rowNumber++
	}

	editedXLSXPath := filepath.Join(tempDir, "excelper-edited.xlsx")
	if err := workbook.SaveAs(editedXLSXPath); err != nil {
		return err
	}
	if err := convertOfficeFile(editedXLSXPath, "xls", tempDir); err != nil {
		return err
	}
	editedXLSPath := filepath.Join(tempDir, "excelper-edited.xls")
	return copyFile(editedXLSPath, path)
}

func composeCellValue(req RuleRequest, value string) string {
	return strings.TrimSpace(req.Prefix) + value + strings.TrimSpace(req.Suffix)
}

func sheetNameByIndex(workbook *excelize.File, index int) (string, error) {
	sheets := workbook.GetSheetList()
	if len(sheets) == 0 {
		return "", fmt.Errorf("工作簿中没有工作表")
	}
	if index < 0 || index >= len(sheets) {
		index = 0
	}
	return sheets[index], nil
}

func convertOfficeFile(inputPath string, format string, outputDir string) error {
	command := exec.Command("libreoffice", "--headless", "--convert-to", format, "--outdir", outputDir, inputPath)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("文件转换失败: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func copyFile(source string, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.Create(target)
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, input)
	return err
}
