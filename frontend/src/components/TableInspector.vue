<script lang="ts" setup>
import { ref, watch } from 'vue'
import { database } from '../wailsjs/go/models'
import { GetTableDetail, ExportTableCSV, ExportTableSQL } from '../wailsjs/go/main/App'

const props = defineProps<{
  tabId: string
  database: string
  table: string
}>()

const detail = ref<database.TableDetail | null>(null)
const loading = ref(false)
const error = ref('')
const activeTab = ref<'columns' | 'indexes' | 'fkeys' | 'ddl'>('columns')

watch(
  () => [props.tabId, props.database, props.table],
  async () => {
    if (!props.tabId || !props.database || !props.table) {
      detail.value = null
      return
    }
    loading.value = true
    error.value = ''
    try {
      detail.value = await GetTableDetail(props.tabId, props.database, props.table)
    } catch (e: any) {
      error.value = e?.message || String(e)
      detail.value = null
    } finally {
      loading.value = false
    }
  },
  { immediate: true }
)

function keyBadge(key: string): string {
  if (key === 'PRI') return 'PK'
  if (key === 'UNI') return 'UQ'
  if (key === 'MUL') return 'IX'
  return ''
}

const exporting = ref(false)

async function exportCSV() {
  exporting.value = true
  try {
    await ExportTableCSV(props.tabId, props.database, props.table)
  } catch (e: any) {
    console.error('Export failed:', e)
  } finally {
    exporting.value = false
  }
}

async function exportSQL() {
  exporting.value = true
  try {
    await ExportTableSQL(props.tabId, props.database, props.table)
  } catch (e: any) {
    console.error('Export failed:', e)
  } finally {
    exporting.value = false
  }
}
</script>

<template>
  <div class="inspector">
    <div class="inspector-header">
      <span class="inspector-title">
        <span class="db-name">{{ database }}.</span>{{ table }}
      </span>
      <span class="export-btns">
        <button class="export-btn" @click="exportCSV" :disabled="exporting" title="Export table to CSV">CSV</button>
        <button class="export-btn" @click="exportSQL" :disabled="exporting" title="Export table to SQL">SQL</button>
      </span>
    </div>

    <div class="inspector-tabs">
      <span
        class="inspector-tab"
        :class="{ active: activeTab === 'columns' }"
        @click="activeTab = 'columns'"
      >Columns</span>
      <span
        class="inspector-tab"
        :class="{ active: activeTab === 'indexes' }"
        @click="activeTab = 'indexes'"
      >Indexes</span>
      <span
        class="inspector-tab"
        :class="{ active: activeTab === 'fkeys' }"
        @click="activeTab = 'fkeys'"
      >Foreign Keys</span>
      <span
        class="inspector-tab"
        :class="{ active: activeTab === 'ddl' }"
        @click="activeTab = 'ddl'"
      >DDL</span>
    </div>

    <div v-if="loading" class="inspector-status">Loading...</div>
    <div v-else-if="error" class="inspector-status error">{{ error }}</div>
    <div v-else-if="!detail" class="inspector-status">Select a table to inspect</div>

    <div v-else class="inspector-content">
      <!-- Columns -->
      <table v-if="activeTab === 'columns'" class="data-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Null</th>
            <th>Key</th>
            <th>Default</th>
            <th>Extra</th>
            <th>Charset</th>
            <th>Collation</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="col in detail.columns" :key="col.name">
            <td class="mono col-name">{{ col.name }}</td>
            <td class="mono">{{ col.columnType }}</td>
            <td>{{ col.nullable ? 'YES' : 'NO' }}</td>
            <td><span v-if="keyBadge(col.key)" class="key-badge" :class="col.key">{{ keyBadge(col.key) }}</span></td>
            <td class="mono">{{ col.default ?? 'NULL' }}</td>
            <td>{{ col.extra }}</td>
            <td>{{ col.charSet ?? '-' }}</td>
            <td>{{ col.collation ?? '-' }}</td>
          </tr>
        </tbody>
      </table>

      <!-- Indexes -->
      <table v-if="activeTab === 'indexes'" class="data-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Columns</th>
            <th>Unique</th>
            <th>Type</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="idx in detail.indexes" :key="idx.name">
            <td class="mono">{{ idx.name }}</td>
            <td class="mono">{{ idx.columns }}</td>
            <td>{{ idx.unique ? 'YES' : 'NO' }}</td>
            <td>{{ idx.type }}</td>
          </tr>
          <tr v-if="detail.indexes.length === 0">
            <td colspan="4" class="empty-cell">No indexes</td>
          </tr>
        </tbody>
      </table>

      <!-- Foreign Keys -->
      <table v-if="activeTab === 'fkeys'" class="data-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Column</th>
            <th>References</th>
            <th>On Update</th>
            <th>On Delete</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="fk in detail.foreignKeys" :key="fk.name + fk.column">
            <td class="mono">{{ fk.name }}</td>
            <td class="mono">{{ fk.column }}</td>
            <td class="mono">{{ fk.refTable }}.{{ fk.refColumn }}</td>
            <td>{{ fk.updateRule }}</td>
            <td>{{ fk.deleteRule }}</td>
          </tr>
          <tr v-if="detail.foreignKeys.length === 0">
            <td colspan="5" class="empty-cell">No foreign keys</td>
          </tr>
        </tbody>
      </table>

      <!-- DDL -->
      <div v-if="activeTab === 'ddl'" class="ddl-container">
        <pre class="ddl-code mono">{{ detail.createSql }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.inspector {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.inspector-header {
  padding: 0.5rem 0.75rem;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.inspector-title {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
}

.export-btns {
  display: flex;
  gap: 2px;
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

.db-name {
  color: var(--text-muted);
  font-weight: 400;
}

.inspector-tabs {
  display: flex;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.inspector-tab {
  padding: 0.35rem 0.85rem;
  font-size: 0.8rem;
  color: var(--text-muted);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  user-select: none;
}

.inspector-tab:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
}

.inspector-tab.active {
  color: var(--text-primary);
  border-bottom-color: var(--accent);
}

.inspector-status {
  padding: 1rem;
  font-size: 0.8rem;
  color: var(--text-muted);
  text-align: center;
  font-style: italic;
}

.inspector-status.error {
  color: var(--error);
  font-style: normal;
}

.inspector-content {
  flex: 1;
  overflow: auto;
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
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 1;
}

.data-table td {
  padding: 0.3rem 0.6rem;
  border-bottom: 1px solid var(--border);
  color: var(--text-secondary);
}

.data-table tr:hover td {
  background: var(--bg-hover);
}

.col-name {
  color: var(--text-primary);
  font-weight: 600;
}

.key-badge {
  font-size: 0.7rem;
  font-weight: 600;
  padding: 1px 4px;
  border-radius: 2px;
  background: var(--bg-input);
}

.key-badge.PRI {
  color: var(--warning);
  background: rgba(224, 175, 104, 0.15);
}

.key-badge.UNI {
  color: var(--accent);
  background: rgba(122, 162, 247, 0.15);
}

.key-badge.MUL {
  color: var(--text-muted);
}

.empty-cell {
  text-align: center;
  color: var(--text-muted);
  font-style: italic;
}

.ddl-container {
  padding: 0.5rem;
}

.ddl-code {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 0.75rem;
  font-size: 0.8rem;
  line-height: 1.5;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-word;
  overflow: auto;
}
</style>
