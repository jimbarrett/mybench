<script lang="ts" setup>
import { ref, computed } from 'vue'
import { database } from '../wailsjs/go/models'
import { ExportResultsCSV, ExportResultsSQL } from '../wailsjs/go/main/App'

const props = defineProps<{
  results?: database.QueryResult[] | null
  loading?: boolean
}>()

const activePanel = ref<'results' | 'messages'>('results')
const sortColumn = ref<number | null>(null)
const sortAsc = ref(true)

// Use the last SELECT result for the results tab
const lastSelectResult = computed(() => {
  if (!props.results) return null
  for (let i = props.results.length - 1; i >= 0; i--) {
    if (props.results[i].isSelect && props.results[i].columns?.length) {
      return props.results[i]
    }
  }
  return null
})

// All results as messages (for Messages tab)
const messages = computed(() => {
  if (!props.results) return []
  return props.results.map((r, i) => {
    if (r.error) return { type: 'error' as const, text: r.error, duration: r.duration }
    if (r.isSelect) return { type: 'info' as const, text: `${r.rowCount} row(s) returned`, duration: r.duration }
    return { type: 'success' as const, text: `${r.affectedRows} row(s) affected`, duration: r.duration }
  })
})

const sortedRows = computed(() => {
  const result = lastSelectResult.value
  if (!result?.rows) return []
  if (sortColumn.value === null) return result.rows

  const col = sortColumn.value
  const dir = sortAsc.value ? 1 : -1

  return [...result.rows].sort((a, b) => {
    const va = a[col] ?? ''
    const vb = b[col] ?? ''
    // Try numeric sort
    const na = Number(va)
    const nb = Number(vb)
    if (!isNaN(na) && !isNaN(nb)) return (na - nb) * dir
    return va.localeCompare(vb) * dir
  })
})

function toggleSort(colIndex: number) {
  if (sortColumn.value === colIndex) {
    sortAsc.value = !sortAsc.value
  } else {
    sortColumn.value = colIndex
    sortAsc.value = true
  }
}

function sortIndicator(colIndex: number): string {
  if (sortColumn.value !== colIndex) return ''
  return sortAsc.value ? ' ▲' : ' ▼'
}

function copyCell(value: string) {
  navigator.clipboard.writeText(value)
}

function copyRow(row: string[]) {
  navigator.clipboard.writeText(row.join('\t'))
}

const exporting = ref(false)

async function exportCSV() {
  const result = lastSelectResult.value
  if (!result?.columns || !result?.rows) return
  exporting.value = true
  try {
    const path = await ExportResultsCSV(result.columns, result.rows)
    if (path) console.log('Exported to', path)
  } catch (e: any) {
    console.error('Export failed:', e)
  } finally {
    exporting.value = false
  }
}

async function exportSQL() {
  const result = lastSelectResult.value
  if (!result?.columns || !result?.rows) return
  exporting.value = true
  try {
    const path = await ExportResultsSQL('table_name', result.columns, result.rows)
    if (path) console.log('Exported to', path)
  } catch (e: any) {
    console.error('Export failed:', e)
  } finally {
    exporting.value = false
  }
}
</script>

<template>
  <div class="results-area">
    <div class="results-tabs">
      <span
        class="results-tab"
        :class="{ active: activePanel === 'results' }"
        @click="activePanel = 'results'"
      >
        Results
        <span v-if="lastSelectResult" class="results-count">({{ lastSelectResult.rowCount }})</span>
      </span>
      <span
        class="results-tab"
        :class="{ active: activePanel === 'messages' }"
        @click="activePanel = 'messages'"
      >
        Messages
        <span v-if="messages.some(m => m.type === 'error')" class="error-dot" />
      </span>
      <div class="results-tabs-spacer" />
      <span v-if="lastSelectResult" class="export-btns">
        <button class="export-btn" @click="exportCSV" :disabled="exporting" title="Export results to CSV">CSV</button>
        <button class="export-btn" @click="exportSQL" :disabled="exporting" title="Export results to SQL">SQL</button>
      </span>
      <span v-if="lastSelectResult" class="results-meta">
        {{ lastSelectResult.rowCount }} rows | {{ lastSelectResult.duration }}
      </span>
    </div>

    <div class="results-content">
      <div v-if="loading" class="empty-state">Executing query...</div>

      <!-- Results Tab -->
      <div v-else-if="activePanel === 'results'">
        <div v-if="!lastSelectResult" class="empty-state">
          Run a query to see results
        </div>
        <div v-else class="table-wrapper">
          <table class="data-table">
            <thead>
              <tr>
                <th class="row-num">#</th>
                <th
                  v-for="(col, i) in lastSelectResult.columns"
                  :key="i"
                  @click="toggleSort(i)"
                  class="sortable"
                >
                  {{ col }}{{ sortIndicator(i) }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, ri) in sortedRows"
                :key="ri"
                @dblclick="copyRow(row)"
              >
                <td class="row-num">{{ ri + 1 }}</td>
                <td
                  v-for="(cell, ci) in row"
                  :key="ci"
                  :class="{ null: cell === 'NULL' }"
                  class="mono"
                  @dblclick.stop="copyCell(cell)"
                >
                  {{ cell }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Messages Tab -->
      <div v-else-if="activePanel === 'messages'">
        <div v-if="messages.length === 0" class="empty-state">No messages</div>
        <div v-else class="messages-list">
          <div
            v-for="(msg, i) in messages"
            :key="i"
            class="message-item"
            :class="msg.type"
          >
            <span class="message-text">{{ msg.text }}</span>
            <span class="message-duration" v-if="msg.duration">{{ msg.duration }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.results-area {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.results-tabs {
  display: flex;
  align-items: center;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.results-tab {
  padding: 0.35rem 0.85rem;
  font-size: 0.8rem;
  color: var(--text-muted);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  user-select: none;
  display: flex;
  align-items: center;
  gap: 0.3rem;
}

.results-tab:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
}

.results-tab.active {
  color: var(--text-primary);
  border-bottom-color: var(--accent);
}

.results-count {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.error-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--error);
}

.results-tabs-spacer {
  flex: 1;
}

.export-btns {
  display: flex;
  gap: 2px;
  margin-right: 0.5rem;
}

.export-btn {
  padding: 1px 6px;
  font-size: 0.75rem;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  border-radius: 2px;
  cursor: pointer;
}

.export-btn:hover {
  color: var(--accent);
  border-color: var(--accent-dim);
}

.export-btn:disabled {
  opacity: 0.5;
  cursor: default;
}

.results-meta {
  font-size: 0.7rem;
  color: var(--text-muted);
  padding-right: 0.75rem;
}

.results-content {
  flex: 1;
  overflow: auto;
  background: var(--bg-primary);
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 60px;
  color: var(--text-muted);
  font-size: 0.85rem;
  font-style: italic;
}

.table-wrapper {
  overflow: auto;
  height: 100%;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8rem;
}

.data-table th {
  text-align: left;
  padding: 0.35rem 0.6rem;
  background: var(--bg-secondary);
  color: var(--text-muted);
  font-weight: 600;
  font-size: 0.75rem;
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 1;
  white-space: nowrap;
}

.data-table th.sortable {
  cursor: pointer;
  user-select: none;
}

.data-table th.sortable:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.data-table td {
  padding: 0.25rem 0.6rem;
  border-bottom: 1px solid var(--border);
  color: var(--text-secondary);
  white-space: nowrap;
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.data-table tr:hover td {
  background: var(--bg-hover);
}

.data-table td.null {
  color: var(--text-muted);
  font-style: italic;
}

.row-num {
  color: var(--text-muted);
  font-size: 0.7rem;
  text-align: right;
  padding-right: 0.5rem;
  width: 40px;
  min-width: 40px;
}

th.row-num {
  cursor: default;
}

/* Messages */
.messages-list {
  padding: 0.5rem;
}

.message-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.35rem 0.6rem;
  border-radius: 4px;
  margin-bottom: 0.25rem;
  font-size: 0.8rem;
}

.message-item.info {
  color: var(--text-secondary);
}

.message-item.success {
  color: var(--success);
  background: rgba(158, 206, 106, 0.08);
}

.message-item.error {
  color: var(--error);
  background: rgba(247, 118, 142, 0.08);
}

.message-duration {
  font-size: 0.7rem;
  color: var(--text-muted);
  flex-shrink: 0;
  margin-left: 1rem;
}
</style>
