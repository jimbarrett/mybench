<script lang="ts" setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { database } from './wailsjs/go/models'
import {
  HasMasterPassword,
  SetMasterPassword,
  UnlockVault,
  ExecuteQuery,
  ExplainQuery,
  CancelQuery,
  GetSchemaCompletions,
  ImportSQL,
} from './wailsjs/go/main/App'
import MasterPasswordDialog from './components/MasterPasswordDialog.vue'
import Sidebar from './components/Sidebar.vue'
import TabBar from './components/TabBar.vue'
import QueryEditor from './components/QueryEditor.vue'
import ResultsTable from './components/ResultsTable.vue'
import StatusBar from './components/StatusBar.vue'
import TableInspector from './components/TableInspector.vue'
import UserInspector from './components/UserInspector.vue'
import CreateUserDialog from './components/CreateUserDialog.vue'
import ImportCSVDialog from './components/ImportCSVDialog.vue'

const unlocked = ref(false)
const needsNewPassword = ref(false)
const showPasswordDialog = ref(true)
const loading = ref(true)

// Active connection state
const activeTabId = ref('')
const connectedName = ref('')

// Query results
const queryResults = ref<database.QueryResult[] | null>(null)
const queryRunning = ref(false)

// Schema completions for editor autocomplete
const schemaCompletions = ref<Record<string, string[]> | null>(null)

// Table inspector state
const inspectedTabId = ref('')
const inspectedDb = ref('')
const inspectedTable = ref('')
const showInspector = ref(false)

// User inspector state
const inspectedUser = ref('')
const inspectedHost = ref('')
const showUserInspector = ref(false)
const showCreateUser = ref(false)

// Import state
const showImportCSV = ref(false)
const importSQLRunning = ref(false)

// Resizable panels
const sidebarWidth = ref(parseInt(localStorage.getItem('mybench:sidebarWidth') || '260'))
const editorHeight = ref(parseInt(localStorage.getItem('mybench:editorHeight') || '0')) // 0 = 50%
const resizing = ref<'sidebar' | 'editor' | null>(null)
const workspaceRef = ref<HTMLElement | null>(null)

// Toast notifications
const toasts = ref<{ id: number; message: string; type: 'error' | 'success' | 'info' }[]>([])
let toastId = 0

function addToast(message: string, type: 'error' | 'success' | 'info' = 'error') {
  const id = ++toastId
  toasts.value.push({ id, message, type })
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }, 5000)
}

function dismissToast(id: number) {
  toasts.value = toasts.value.filter(t => t.id !== id)
}

const editorRef = ref<InstanceType<typeof QueryEditor> | null>(null)
const sidebarRef = ref<InstanceType<typeof Sidebar> | null>(null)

// --- Resize handlers ---
function startSidebarResize(e: MouseEvent) {
  e.preventDefault()
  resizing.value = 'sidebar'
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

function startEditorResize(e: MouseEvent) {
  e.preventDefault()
  resizing.value = 'editor'
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'
}

function onMouseMove(e: MouseEvent) {
  if (resizing.value === 'sidebar') {
    const w = Math.max(180, Math.min(500, e.clientX))
    sidebarWidth.value = w
  } else if (resizing.value === 'editor' && workspaceRef.value) {
    const rect = workspaceRef.value.getBoundingClientRect()
    const h = Math.max(80, Math.min(rect.height - 80, e.clientY - rect.top))
    editorHeight.value = h
  }
}

function stopResize() {
  if (resizing.value === 'sidebar') {
    localStorage.setItem('mybench:sidebarWidth', String(sidebarWidth.value))
  } else if (resizing.value === 'editor') {
    localStorage.setItem('mybench:editorHeight', String(editorHeight.value))
  }
  resizing.value = null
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

// --- Global keyboard shortcuts ---
function handleGlobalKeydown(e: KeyboardEvent) {
  // Ctrl+L / Cmd+L: focus editor
  if ((e.ctrlKey || e.metaKey) && e.key === 'l') {
    e.preventDefault()
    editorRef.value?.focus()
    return
  }
  // Escape: blur active element (return focus from editor to app)
  if (e.key === 'Escape') {
    const active = document.activeElement as HTMLElement
    if (active && active !== document.body) {
      active.blur()
    }
    return
  }
}

// --- State persistence ---
function saveEditorState() {
  const content = editorRef.value?.getContent()
  if (content !== undefined) {
    localStorage.setItem('mybench:editorContent', content)
  }
  if (connectedName.value) {
    localStorage.setItem('mybench:lastConnection', connectedName.value)
  }
}

function restoreEditorState() {
  const content = localStorage.getItem('mybench:editorContent')
  if (content) {
    editorRef.value?.setContent(content)
  }
}

onMounted(async () => {
  document.addEventListener('keydown', handleGlobalKeydown)

  // Auto-save editor state periodically
  const saveInterval = setInterval(saveEditorState, 5000)
  ;(window as any).__mybenchSaveInterval = saveInterval

  try {
    const hasPw = await HasMasterPassword()
    needsNewPassword.value = !hasPw
    showPasswordDialog.value = true
  } catch (e) {
    console.error('Failed to check master password:', e)
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleGlobalKeydown)
  clearInterval((window as any).__mybenchSaveInterval)
  saveEditorState()
})

async function handlePassword(password: string) {
  try {
    if (needsNewPassword.value) {
      await SetMasterPassword(password)
      unlocked.value = true
      showPasswordDialog.value = false
    } else {
      const ok = await UnlockVault(password)
      if (ok) {
        unlocked.value = true
        showPasswordDialog.value = false
      }
    }
  } catch (e: any) {
    addToast(e?.message || 'Password error', 'error')
  }
}

async function onConnected(profileId: string, tabId: string) {
  activeTabId.value = tabId
  connectedName.value = profileId
  localStorage.setItem('mybench:lastConnection', profileId)

  // Restore editor content on first connect
  restoreEditorState()

  // Fetch schema for autocomplete in the background
  try {
    schemaCompletions.value = await GetSchemaCompletions(tabId)
  } catch (e: any) {
    addToast('Failed to load autocomplete data', 'error')
  }
}

function onDisconnected(profileId: string) {
  if (connectedName.value === profileId) {
    activeTabId.value = ''
    connectedName.value = ''
    showInspector.value = false
    showUserInspector.value = false
    showCreateUser.value = false
    queryResults.value = null
    schemaCompletions.value = null
  }
}

function onSelectTable(tabId: string, db: string, table: string) {
  inspectedTabId.value = tabId
  inspectedDb.value = db
  inspectedTable.value = table
  showInspector.value = true
  showUserInspector.value = false
}

function onQueryTable(tabId: string, db: string, table: string) {
  const sql = 'SELECT * FROM `' + db + '`.`' + table + '` LIMIT 25;'
  editorRef.value?.setContent(sql)
  handleExecute(sql)
}

function onSelectUser(tabId: string, user: string, host: string) {
  inspectedTabId.value = tabId
  inspectedUser.value = user
  inspectedHost.value = host
  showUserInspector.value = true
  showInspector.value = false
}

function onCreateUser(tabId: string) {
  showCreateUser.value = true
}

function onUserCreated() {
  showCreateUser.value = false
  sidebarRef.value?.refreshUsers()
}

function onUserChanged() {
  showUserInspector.value = false
  sidebarRef.value?.refreshUsers()
}

function openImportCSV() {
  if (!activeTabId.value) return
  showImportCSV.value = true
}

function onImportCSVDone(rows: number) {
  showImportCSV.value = false
}

async function handleImportSQL() {
  if (!activeTabId.value) return
  importSQLRunning.value = true
  try {
    const stmts = await ImportSQL(activeTabId.value)
    if (stmts > 0) {
      queryResults.value = [{ error: '', isSelect: false, affectedRows: stmts, rowCount: 0, columns: [], rows: [], duration: '' } as any]
    }
  } catch (e: any) {
    queryResults.value = [{ error: e?.message || String(e) } as any]
  } finally {
    importSQLRunning.value = false
  }
}

async function handleExecute(sql: string) {
  if (!activeTabId.value) return
  showInspector.value = false
  showUserInspector.value = false
  queryRunning.value = true
  editorRef.value?.setRunning(true)
  try {
    queryResults.value = await ExecuteQuery(activeTabId.value, sql)
  } catch (e: any) {
    queryResults.value = [{ error: e?.message || String(e) } as database.QueryResult]
  } finally {
    queryRunning.value = false
    editorRef.value?.setRunning(false)
  }
}

async function handleExplain(sql: string) {
  if (!activeTabId.value) return
  showInspector.value = false
  showUserInspector.value = false
  queryRunning.value = true
  editorRef.value?.setRunning(true)
  try {
    const result = await ExplainQuery(activeTabId.value, sql)
    queryResults.value = [result]
  } catch (e: any) {
    queryResults.value = [{ error: e?.message || String(e) } as database.QueryResult]
  } finally {
    queryRunning.value = false
    editorRef.value?.setRunning(false)
  }
}

async function handleCancel() {
  if (!activeTabId.value) return
  try {
    await CancelQuery(activeTabId.value)
  } catch (e: any) {
    addToast('Cancel failed: ' + (e?.message || String(e)), 'error')
  }
}
</script>

<template>
  <div v-if="loading" class="loading">Loading...</div>

  <MasterPasswordDialog
    v-else-if="showPasswordDialog"
    :isNew="needsNewPassword"
    @submit="handlePassword"
  />

  <div v-else class="app-layout">
    <Sidebar
      ref="sidebarRef"
      :style="{ width: sidebarWidth + 'px', minWidth: sidebarWidth + 'px' }"
      @connected="onConnected"
      @disconnected="onDisconnected"
      @select-table="onSelectTable"
      @query-table="onQueryTable"
      @select-user="onSelectUser"
      @create-user="onCreateUser"
    />
    <div
      class="resize-handle-v"
      @mousedown="startSidebarResize"
    />
    <div class="main-area">
      <TabBar />
      <div class="workspace" ref="workspaceRef">
        <QueryEditor
          ref="editorRef"
          :tabId="activeTabId"
          :schema="schemaCompletions"
          :style="editorHeight ? { flex: 'none', height: editorHeight + 'px' } : {}"
          @execute="handleExecute"
          @explain="handleExplain"
          @cancel="handleCancel"
          @import-csv="openImportCSV"
          @import-sql="handleImportSQL"
        />
        <div
          class="resize-handle-h"
          @mousedown="startEditorResize"
        />
        <div class="results-wrapper">
          <TableInspector
            v-if="showInspector"
            :tabId="inspectedTabId"
            :database="inspectedDb"
            :table="inspectedTable"
          />
          <UserInspector
            v-else-if="showUserInspector"
            :tabId="inspectedTabId"
            :user="inspectedUser"
            :host="inspectedHost"
            @user-changed="onUserChanged"
          />
          <ResultsTable
            v-else
            :results="queryResults"
            :loading="queryRunning"
          />
        </div>
      </div>
      <StatusBar :connected="!!connectedName" />
    </div>

    <!-- Toasts -->
    <div class="toast-container">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="toast"
        :class="toast.type"
        @click="dismissToast(toast.id)"
      >
        {{ toast.message }}
      </div>
    </div>

    <CreateUserDialog
      v-if="showCreateUser"
      :tabId="activeTabId"
      @created="onUserCreated"
      @cancel="showCreateUser = false"
    />

    <ImportCSVDialog
      v-if="showImportCSV"
      :tabId="activeTabId"
      @done="onImportCSVDone"
      @cancel="showImportCSV = false"
    />
  </div>
</template>

<style scoped>
.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-muted);
  font-size: 0.9rem;
}

.app-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.workspace {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.results-wrapper {
  flex: 1;
  overflow: hidden;
}

/* Resize handles */
.resize-handle-v {
  width: 4px;
  cursor: col-resize;
  background: transparent;
  flex-shrink: 0;
  position: relative;
  z-index: 10;
}

.resize-handle-v:hover,
.resize-handle-v:active {
  background: var(--accent-dim);
}

.resize-handle-h {
  height: 4px;
  cursor: row-resize;
  background: transparent;
  flex-shrink: 0;
  position: relative;
  z-index: 10;
}

.resize-handle-h:hover,
.resize-handle-h:active {
  background: var(--accent-dim);
}

/* Toast notifications */
.toast-container {
  position: fixed;
  bottom: 2rem;
  right: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  z-index: 200;
  pointer-events: none;
}

.toast {
  padding: 0.5rem 0.85rem;
  border-radius: 6px;
  font-size: 0.8rem;
  max-width: 380px;
  word-break: break-word;
  cursor: pointer;
  pointer-events: auto;
  animation: toast-in 0.2s ease-out;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.toast.error {
  background: rgba(247, 118, 142, 0.15);
  border: 1px solid var(--error);
  color: var(--error);
}

.toast.success {
  background: rgba(158, 206, 106, 0.15);
  border: 1px solid var(--success);
  color: var(--success);
}

.toast.info {
  background: rgba(122, 162, 247, 0.15);
  border: 1px solid var(--accent);
  color: var(--accent);
}

@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
