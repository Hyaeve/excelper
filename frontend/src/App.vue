<template>
  <div class="page-shell">
    <div class="panel source-panel">
      <div class="panel-header">
        <div class="title-wrap">
          <h2>待命名</h2>
          <span class="badge">{{ files.length }} 项</span>
        </div>
        <div class="toolbar">
          <button class="icon-button" @click="loadFiles">⟳</button>
          <select v-model="selectedFile" @change="handleFileChange">
            <option value="">选择挂载目录中的 xls 文件</option>
            <option v-for="file in files" :key="file" :value="file">{{ file }}</option>
          </select>
        </div>
      </div>
      <div class="panel-body">
        <div v-if="sourceSheets.length" class="sheet-preview">
          <div v-for="sheet in sourceSheets" :key="sheet.name" class="sheet-card">
            <div class="sheet-title">{{ sheet.name }}</div>
            <div class="table-wrap source-table-wrap" @scroll="handleSourceScroll">
              <table>
                <tbody>
                  <tr v-for="(row, rowIndex) in sheet.rows" :key="rowIndex">
                    <td v-for="(cell, cellIndex) in row" :key="cellIndex">{{ cell }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <div v-if="sourceLoading" class="loading-state">正在加载...</div>
        </div>
        <div v-else class="empty-state">请选择一个挂载目录内的 xls 文件进行预览</div>
      </div>
    </div>

    <div class="center-bar">
      <div class="rule-card">
        <label>
          <span>填写列</span>
          <input v-model="form.column" placeholder="如 B" />
        </label>
        <label>
          <span>起始行（必填）</span>
          <input v-model.number="form.startRow" type="number" min="1" placeholder="每次录入都必须填写，如 924" />
        </label>
        <label>
          <span>固定前缀</span>
          <input v-model="form.prefix" placeholder="如 25B140-" />
        </label>
        <label>
          <span>固定后缀</span>
          <input v-model="form.suffix" placeholder="如 自交" />
        </label>
        <label>
          <span>录入规则</span>
          <textarea v-model="form.values" rows="7" placeholder="如 51 +58 66 +71 73 76"></textarea>
        </label>
        <button class="action-button preview" @click="previewResult">预览</button>
        <button class="action-button execute" @click="executeResult">执行</button>
        <div class="tip">空格表示下一行；+ 表示先插入空白行再录入；不带 + 则直接录入原表当前行</div>
      </div>
    </div>

    <div class="panel result-panel">
      <div class="panel-header">
        <div class="title-wrap">
          <h2>命名结果</h2>
          <span v-if="preview.generated.length" class="badge primary">{{ preview.generated.length }} 条</span>
        </div>
        <button class="icon-button" @click="previewResult">↻</button>
      </div>
      <div class="panel-body">
        <div v-if="preview.generated.length" class="result-content">
          <div class="meta-card">
            <div><strong>备份文件：</strong>{{ preview.backupName }}</div>
            <div><strong>输出文件：</strong>{{ preview.outputName }}（覆盖原文件）</div>
          </div>
          <div class="table-wrap generated-wrap">
            <table>
              <thead>
                <tr>
                  <th>行号</th>
                  <th>列</th>
                  <th>插入</th>
                  <th>预览值</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in preview.generated" :key="`${item.row}-${item.value}`">
                  <td>{{ item.row }}</td>
                  <td>{{ item.column }}</td>
                  <td>{{ item.inserted ? '是' : '否' }}</td>
                  <td>{{ item.value }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div v-else class="empty-state">点击中间的预览按钮后显示录入预览结果</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'

const previewLimit = 60
const files = ref([])
const selectedFile = ref('')
const sourceSheets = ref([])
const sourceOffset = ref(0)
const sourceHasMore = ref(false)
const sourceLoading = ref(false)
const preview = reactive({
  generated: [],
  outputName: '',
  backupName: ''
})
const form = reactive({
  column: 'B',
  startRow: null,
  prefix: '',
  suffix: '',
  values: ''
})

async function loadFiles() {
  const response = await fetch('/api/files')
  const data = await response.json()
  files.value = data.files || []
}

async function handleFileChange() {
  if (!selectedFile.value) {
    sourceSheets.value = []
    sourceOffset.value = 0
    sourceHasMore.value = false
    return
  }
  await loadSourcePreview(true)
}

async function loadSourcePreview(reset = false) {
  if (!selectedFile.value || sourceLoading.value) {
    return
  }
  if (!reset && !sourceHasMore.value) {
    return
  }
  sourceLoading.value = true
  const offset = reset ? 0 : sourceOffset.value
  try {
    const response = await fetch(`/api/preview?file=${encodeURIComponent(selectedFile.value)}&offset=${offset}&limit=${previewLimit}`)
    const data = await response.json()
    if (!response.ok) {
      alert(data.error || '加载预览失败')
      return
    }
    const nextSheets = data.sheets || []
    if (reset) {
      sourceSheets.value = nextSheets
    } else {
      sourceSheets.value = sourceSheets.value.map((sheet) => {
        const nextSheet = nextSheets.find((item) => item.name === sheet.name)
        if (!nextSheet) {
          return sheet
        }
        return { ...nextSheet, rows: [...sheet.rows, ...nextSheet.rows] }
      })
    }
    sourceOffset.value = offset + previewLimit
    sourceHasMore.value = nextSheets.some((sheet) => sheet.hasMore)
  } finally {
    sourceLoading.value = false
  }
}

function handleSourceScroll(event) {
  const target = event.target
  if (target.scrollTop + target.clientHeight >= target.scrollHeight - 24) {
    loadSourcePreview(false)
  }
}

function validateForm() {
  if (!selectedFile.value) {
    alert('请先选择挂载目录中的 xls 文件')
    return false
  }
  if (!form.startRow || form.startRow < 1) {
    alert('每次录入语法操作都必须定义从表格第几行开始')
    return false
  }
  return true
}

async function previewResult() {
  if (!validateForm()) {
    return
  }
  const response = await fetch('/api/preview', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...form, fileName: selectedFile.value, sheetIndex: 0 })
  })
  const data = await response.json()
  if (!response.ok) {
    alert(data.error || '预览失败')
    return
  }
  preview.generated = data.generated || []
  preview.outputName = data.outputName || ''
  preview.backupName = data.backupName || ''
}

async function executeResult() {
  if (!validateForm()) {
    return
  }
  const response = await fetch('/api/execute', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...form, fileName: selectedFile.value, sheetIndex: 0 })
  })
  const data = await response.json()
  if (!response.ok) {
    alert(data.error || '执行失败')
    return
  }
  alert(`${data.message}\n备份文件: ${data.backupName}\n输出文件: ${data.outputName}`)
  await loadFiles()
  await loadSourcePreview(true)
}

onMounted(loadFiles)
</script>
