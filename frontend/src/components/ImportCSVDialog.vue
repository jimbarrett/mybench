<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { main, database } from '../wailsjs/go/models'
import {
  ImportOpenCSV,
  ImportCSV,
  GetTableColumns,
  GetDatabases,
  GetTables,
} from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

const props = defineProps<{
  tabId: string
}>()

const emit = defineEmits<{
  (e: 'done', rows: number): void
  (e: 'cancel'): void
}>()

// Steps: pick-file -> map-columns -> importing -> done
const step = ref<'pick-file' | 'map-columns' | 'importing' | 'done'>('pick-file')

// File preview data
const preview = ref<main.CSVImportPreview | null>(null)
const error = ref('')

// Target table selection
const databases = ref<database.DatabaseInfo[]>([])
const tables = ref<database.TableInfo[]>([])
const selectedDb = ref('')
const selectedTable = ref('')
const tableColumns = ref<string[]>([])

// Column mappings: index = CSV column index, value = DB column name (or '' to skip)
const mappings = ref<string[]>([])

// Import progress
const importProgress = ref({ current: 0, total: 0 })
const importResult = ref({ rows: 0, error: '' })

let cleanupProgress: (() => void) | null = null

onMounted(async () => {
  // Load databases for target selection
  try {
    databases.value = await GetDatabases(props.tabId) || []
  } catch (e) {
    console.error('Failed to load databases:', e)
  }

  // Listen for progress events
  cleanupProgress = EventsOn('import-progress', (data: any) => {
    importProgress.value = { current: data.current || 0, total: data.total || 0 }
  })
})

onUnmounted(() => {
  if (cleanupProgress) cleanupProgress()
})

async function pickFile() {
  error.value = ''
  try {
    const result = await ImportOpenCSV()
    if (!result) return // cancelled
    preview.value = result
    // Initialize mappings (all empty)
    mappings.value = (result.headers || []).map(() => '')
    step.value = 'map-columns'
  } catch (e: any) {
    error.value = e?.message || String(e)
  }
}

async function onDbChange() {
  if (!selectedDb.value) {
    tables.value = []
    selectedTable.value = ''
    return
  }
  try {
    tables.value = await GetTables(props.tabId, selectedDb.value) || []
  } catch (e) {
    console.error('Failed to load tables:', e)
  }
}

async function onTableChange() {
  if (!selectedDb.value || !selectedTable.value) {
    tableColumns.value = []
    return
  }
  try {
    tableColumns.value = await GetTableColumns(props.tabId, selectedDb.value, selectedTable.value) || []
    // Auto-map by matching header names (case-insensitive)
    if (preview.value) {
      mappings.value = preview.value.headers.map(h => {
        const match = tableColumns.value.find(c => c.toLowerCase() === h.toLowerCase())
        return match || ''
      })
    }
  } catch (e) {
    console.error('Failed to load columns:', e)
  }
}

const validMappings = computed(() => {
  return mappings.value.some(m => m !== '')
})

async function startImport() {
  if (!preview.value || !selectedDb.value || !selectedTable.value) return
  error.value = ''
  step.value = 'importing'
  importProgress.value = { current: 0, total: preview.value.totalRows }

  // Build column mapping array
  const colMappings: database.ColumnMapping[] = []
  for (let i = 0; i < mappings.value.length; i++) {
    if (mappings.value[i]) {
      colMappings.push(new database.ColumnMapping({
        csvIndex: i,
        columnName: mappings.value[i],
      }))
    }
  }

  try {
    const rows = await ImportCSV(
      props.tabId,
      selectedDb.value,
      selectedTable.value,
      preview.value.filePath,
      colMappings,
    )
    importResult.value = { rows, error: '' }
    step.value = 'done'
  } catch (e: any) {
    importResult.value = { rows: importProgress.value.current, error: e?.message || String(e) }
    step.value = 'done'
  }
}
</script>

<template>
  <div class="dialog-overlay" @click.self="emit('cancel')">
    <div class="dialog import-dialog">
      <h2 class="dialog-title">Import CSV</h2>

      <!-- Step 1: Pick File -->
      <div v-if="step === 'pick-file'" class="step">
        <p class="step-desc">Select a CSV file to import into a database table.</p>
        <button class="primary" @click="pickFile">Choose CSV File...</button>
        <div v-if="error" class="error-msg">{{ error }}</div>
      </div>

      <!-- Step 2: Map Columns -->
      <div v-if="step === 'map-columns' && preview" class="step">
        <div class="file-info">
          <span class="file-path">{{ preview.filePath }}</span>
          <span class="file-rows">{{ preview.totalRows }} rows</span>
        </div>

        <div class="target-section">
          <div class="form-row">
            <label>Database</label>
            <select v-model="selectedDb" @change="onDbChange">
              <option value="">Select database...</option>
              <option v-for="db in databases" :key="db.name" :value="db.name">{{ db.name }}</option>
            </select>
          </div>
          <div class="form-row">
            <label>Table</label>
            <select v-model="selectedTable" @change="onTableChange" :disabled="!selectedDb">
              <option value="">Select table...</option>
              <option v-for="t in tables" :key="t.name" :value="t.name">{{ t.name }}</option>
            </select>
          </div>
        </div>

        <div v-if="tableColumns.length > 0" class="mapping-section">
          <div class="mapping-header">
            <span>CSV Column</span>
            <span>DB Column</span>
          </div>
          <div v-for="(header, i) in preview.headers" :key="i" class="mapping-row">
            <span class="csv-col mono">{{ header }}</span>
            <select v-model="mappings[i]">
              <option value="">(skip)</option>
              <option v-for="col in tableColumns" :key="col" :value="col">{{ col }}</option>
            </select>
          </div>
        </div>

        <div v-if="preview.sampleRows && preview.sampleRows.length > 0" class="preview-section">
          <div class="preview-label">Preview</div>
          <div class="preview-table-wrap">
            <table class="preview-table">
              <thead>
                <tr>
                  <th v-for="(h, i) in preview.headers" :key="i">{{ h }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, ri) in preview.sampleRows" :key="ri">
                  <td v-for="(cell, ci) in row" :key="ci" class="mono">{{ cell }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-if="error" class="error-msg">{{ error }}</div>

        <div class="dialog-actions">
          <button @click="emit('cancel')">Cancel</button>
          <button
            class="primary"
            @click="startImport"
            :disabled="!validMappings || !selectedTable"
          >Import</button>
        </div>
      </div>

      <!-- Step 3: Importing -->
      <div v-if="step === 'importing'" class="step">
        <div class="progress-section">
          <div class="progress-text">
            Importing... {{ importProgress.current.toLocaleString() }}
            <span v-if="importProgress.total > 0"> / {{ importProgress.total.toLocaleString() }} rows</span>
            <span v-else> rows</span>
          </div>
          <div v-if="importProgress.total > 0" class="progress-bar">
            <div
              class="progress-fill"
              :style="{ width: Math.min(100, (importProgress.current / importProgress.total) * 100) + '%' }"
            />
          </div>
        </div>
      </div>

      <!-- Step 4: Done -->
      <div v-if="step === 'done'" class="step">
        <div v-if="importResult.error" class="result-error">
          <p>Import stopped with error after {{ importResult.rows.toLocaleString() }} rows:</p>
          <p class="error-msg">{{ importResult.error }}</p>
        </div>
        <div v-else class="result-success">
          Successfully imported {{ importResult.rows.toLocaleString() }} rows.
        </div>
        <div class="dialog-actions">
          <button class="primary" @click="emit('done', importResult.rows)">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.import-dialog {
  background: var(--bg-sidebar);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1.25rem;
  width: 560px;
  max-width: 90vw;
  max-height: 80vh;
  overflow-y: auto;
}

.dialog-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 1rem;
}

.step {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.step-desc {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.file-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.4rem 0.6rem;
  background: var(--bg-input);
  border-radius: 4px;
  font-size: 0.8rem;
}

.file-path {
  color: var(--text-primary);
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-rows {
  color: var(--text-muted);
  flex-shrink: 0;
  margin-left: 0.5rem;
}

.target-section {
  display: flex;
  gap: 0.5rem;
}

.form-row {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.form-row label {
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.form-row select {
  padding: 0.35rem 0.5rem;
  font-size: 0.8rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  cursor: pointer;
}

.form-row select:focus {
  border-color: var(--accent);
  outline: none;
}

.mapping-section {
  border: 1px solid var(--border);
  border-radius: 4px;
  overflow: hidden;
}

.mapping-header {
  display: flex;
  padding: 0.3rem 0.6rem;
  background: var(--bg-secondary);
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.mapping-header span {
  flex: 1;
}

.mapping-row {
  display: flex;
  align-items: center;
  padding: 0.25rem 0.6rem;
  border-top: 1px solid var(--border);
}

.mapping-row .csv-col {
  flex: 1;
  font-size: 0.8rem;
  color: var(--text-primary);
}

.mapping-row select {
  flex: 1;
  padding: 0.2rem 0.4rem;
  font-size: 0.8rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 3px;
  color: var(--text-primary);
}

.preview-section {
  margin-top: 0.25rem;
}

.preview-label {
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  margin-bottom: 0.25rem;
}

.preview-table-wrap {
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: 4px;
}

.preview-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.75rem;
}

.preview-table th {
  text-align: left;
  padding: 0.25rem 0.5rem;
  background: var(--bg-secondary);
  color: var(--text-muted);
  font-weight: 600;
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}

.preview-table td {
  padding: 0.2rem 0.5rem;
  border-bottom: 1px solid var(--border);
  color: var(--text-secondary);
  white-space: nowrap;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.error-msg {
  color: var(--error);
  font-size: 0.8rem;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.progress-section {
  padding: 1rem 0;
}

.progress-text {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-bottom: 0.5rem;
}

.progress-bar {
  height: 4px;
  background: var(--bg-input);
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.3s;
  border-radius: 2px;
}

.result-success {
  font-size: 0.9rem;
  color: var(--success);
  padding: 1rem 0;
}

.result-error {
  font-size: 0.85rem;
  color: var(--text-secondary);
  padding: 0.5rem 0;
}

.mono {
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
}
</style>
