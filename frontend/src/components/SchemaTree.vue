<script lang="ts" setup>
import { ref, watch } from 'vue'
import { database } from '../wailsjs/go/models'
import {
  GetDatabases,
  GetTables,
  GetRoutines,
  GetTriggers,
  ListUsers,
} from '../wailsjs/go/main/App'

const props = defineProps<{
  tabId: string
  connected: boolean
}>()

const emit = defineEmits<{
  (e: 'select-table', db: string, table: string): void
  (e: 'query-table', db: string, table: string): void
  (e: 'select-user', user: string, host: string): void
  (e: 'create-user'): void
}>()

interface DbNode {
  name: string
  expanded: boolean
  loading: boolean
  tables: database.TableInfo[]
  routines: database.RoutineInfo[]
  triggers: database.TriggerInfo[]
  tablesExpanded: boolean
  viewsExpanded: boolean
  routinesExpanded: boolean
  triggersExpanded: boolean
}

const databases = ref<DbNode[]>([])
const loading = ref(false)
const error = ref('')

// Users state
const users = ref<database.UserInfo[]>([])
const usersExpanded = ref(false)
const usersLoading = ref(false)
const selectedUser = ref<{ user: string, host: string } | null>(null)

watch(() => props.connected, async (isConnected) => {
  if (isConnected) {
    await loadDatabases()
  } else {
    databases.value = []
    users.value = []
    usersExpanded.value = false
    selectedUser.value = null
  }
}, { immediate: true })

async function loadDatabases() {
  if (!props.tabId) return
  loading.value = true
  error.value = ''
  try {
    const dbs = await GetDatabases(props.tabId)
    databases.value = (dbs || []).map(db => ({
      name: db.name,
      expanded: false,
      loading: false,
      tables: [],
      routines: [],
      triggers: [],
      tablesExpanded: false,
      viewsExpanded: false,
      routinesExpanded: false,
      triggersExpanded: false,
    }))
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

async function toggleDb(db: DbNode) {
  db.expanded = !db.expanded
  if (db.expanded && db.tables.length === 0 && !db.loading) {
    db.loading = true
    try {
      const [tables, routines, triggers] = await Promise.all([
        GetTables(props.tabId, db.name),
        GetRoutines(props.tabId, db.name),
        GetTriggers(props.tabId, db.name),
      ])
      db.tables = tables || []
      db.routines = routines || []
      db.triggers = triggers || []
      db.tablesExpanded = true
    } catch (e: any) {
      console.error('Failed to load schema for', db.name, e)
    } finally {
      db.loading = false
    }
  }
}

function tables(db: DbNode): database.TableInfo[] {
  return db.tables.filter(t => t.type === 'BASE TABLE')
}

function views(db: DbNode): database.TableInfo[] {
  return db.tables.filter(t => t.type === 'VIEW')
}

function selectTable(dbName: string, tableName: string) {
  emit('select-table', dbName, tableName)
}

function queryTable(e: Event, dbName: string, tableName: string) {
  e.stopPropagation()
  emit('query-table', dbName, tableName)
}

async function toggleUsers() {
  usersExpanded.value = !usersExpanded.value
  if (usersExpanded.value && users.value.length === 0) {
    await loadUsers()
  }
}

async function loadUsers() {
  if (!props.tabId) return
  usersLoading.value = true
  try {
    users.value = await ListUsers(props.tabId) || []
  } catch (e: any) {
    console.error('Failed to load users:', e)
  } finally {
    usersLoading.value = false
  }
}

function selectUser(user: string, host: string) {
  selectedUser.value = { user, host }
  emit('select-user', user, host)
}

function handleCreateUser() {
  emit('create-user')
}

async function refreshUsers() {
  users.value = []
  await loadUsers()
}

async function refresh() {
  databases.value = []
  users.value = []
  usersExpanded.value = false
  await loadDatabases()
}

defineExpose({ refresh, refreshUsers })
</script>

<template>
  <div class="schema-tree">
    <div class="tree-header">
      <span class="section-label">Schema</span>
      <button
        v-if="connected"
        class="refresh-btn"
        @click="refresh"
        title="Refresh schema"
      >&#8635;</button>
    </div>

    <div v-if="loading" class="tree-status">Loading...</div>
    <div v-else-if="error" class="tree-status error">{{ error }}</div>
    <div v-else-if="!connected" class="tree-status">Connect to browse schema</div>
    <div v-else-if="databases.length === 0" class="tree-status">No databases found</div>

    <div v-else class="tree-content">
      <!-- Users -->
      <div class="tree-node">
        <div class="tree-row db-row" @click="toggleUsers">
          <span class="arrow" :class="{ expanded: usersExpanded }">&#9654;</span>
          <span class="icon">&#9873;</span>
          <span class="node-name">Users</span>
          <span v-if="usersLoading" class="loading-dot">...</span>
          <span v-else-if="users.length > 0" class="row-count">{{ users.length }}</span>
        </div>
        <div v-if="usersExpanded" class="tree-children">
          <div
            v-for="u in users"
            :key="u.user + '@' + u.host"
            class="tree-row leaf-row"
            :class="{ selected: selectedUser?.user === u.user && selectedUser?.host === u.host }"
            @click="selectUser(u.user, u.host)"
          >
            <span class="icon">&#9679;</span>
            <span class="node-name">'{{ u.user }}'@'{{ u.host }}'</span>
            <span v-if="u.plugin" class="badge">{{ u.plugin }}</span>
          </div>
          <div v-if="users.length === 0 && !usersLoading" class="tree-empty">No users found</div>
          <div class="tree-row leaf-row add-user-row" @click="handleCreateUser">
            <span class="icon">+</span>
            <span class="node-name add-label">New User</span>
          </div>
        </div>
      </div>

      <div v-for="db in databases" :key="db.name" class="tree-node">
        <!-- Database -->
        <div class="tree-row db-row" @click="toggleDb(db)">
          <span class="arrow" :class="{ expanded: db.expanded }">&#9654;</span>
          <span class="icon">&#128450;</span>
          <span class="node-name">{{ db.name }}</span>
          <span v-if="db.loading" class="loading-dot">...</span>
        </div>

        <div v-if="db.expanded" class="tree-children">
          <!-- Tables -->
          <div class="tree-row group-row" @click="db.tablesExpanded = !db.tablesExpanded">
            <span class="arrow" :class="{ expanded: db.tablesExpanded }">&#9654;</span>
            <span class="group-name">Tables ({{ tables(db).length }})</span>
          </div>
          <div v-if="db.tablesExpanded" class="tree-children">
            <div
              v-for="t in tables(db)"
              :key="t.name"
              class="tree-row leaf-row"
              @click="selectTable(db.name, t.name)"
            >
              <span class="icon">&#9638;</span>
              <span class="node-name">{{ t.name }}</span>
              <span class="row-count" v-if="t.rowCount >= 0">{{ t.rowCount.toLocaleString() }}</span>
              <span class="query-btn" title="Query table" @click="queryTable($event, db.name, t.name)">&#9654;</span>
            </div>
            <div v-if="tables(db).length === 0" class="tree-empty">No tables</div>
          </div>

          <!-- Views -->
          <div class="tree-row group-row" @click="db.viewsExpanded = !db.viewsExpanded">
            <span class="arrow" :class="{ expanded: db.viewsExpanded }">&#9654;</span>
            <span class="group-name">Views ({{ views(db).length }})</span>
          </div>
          <div v-if="db.viewsExpanded" class="tree-children">
            <div
              v-for="v in views(db)"
              :key="v.name"
              class="tree-row leaf-row"
              @click="selectTable(db.name, v.name)"
            >
              <span class="icon">&#9671;</span>
              <span class="node-name">{{ v.name }}</span>
              <span class="query-btn" title="Query view" @click="queryTable($event, db.name, v.name)">&#9654;</span>
            </div>
            <div v-if="views(db).length === 0" class="tree-empty">No views</div>
          </div>

          <!-- Routines -->
          <div class="tree-row group-row" @click="db.routinesExpanded = !db.routinesExpanded">
            <span class="arrow" :class="{ expanded: db.routinesExpanded }">&#9654;</span>
            <span class="group-name">Routines ({{ db.routines.length }})</span>
          </div>
          <div v-if="db.routinesExpanded" class="tree-children">
            <div v-for="r in db.routines" :key="r.name" class="tree-row leaf-row">
              <span class="icon">&#9881;</span>
              <span class="node-name">{{ r.name }}</span>
              <span class="badge">{{ r.type === 'PROCEDURE' ? 'proc' : 'func' }}</span>
            </div>
            <div v-if="db.routines.length === 0" class="tree-empty">No routines</div>
          </div>

          <!-- Triggers -->
          <div class="tree-row group-row" @click="db.triggersExpanded = !db.triggersExpanded">
            <span class="arrow" :class="{ expanded: db.triggersExpanded }">&#9654;</span>
            <span class="group-name">Triggers ({{ db.triggers.length }})</span>
          </div>
          <div v-if="db.triggersExpanded" class="tree-children">
            <div v-for="t in db.triggers" :key="t.name" class="tree-row leaf-row">
              <span class="icon">&#9889;</span>
              <span class="node-name">{{ t.name }}</span>
              <span class="badge">{{ t.timing }} {{ t.event }}</span>
            </div>
            <div v-if="db.triggers.length === 0" class="tree-empty">No triggers</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.schema-tree {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tree-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.25rem 1rem;
}

.section-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
}

.refresh-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 2px 4px;
  font-size: 0.85rem;
  border-radius: 2px;
}

.refresh-btn:hover {
  color: var(--accent);
  background: var(--bg-input);
}

.tree-status {
  padding: 1rem;
  font-size: 0.8rem;
  color: var(--text-muted);
  text-align: center;
  font-style: italic;
}

.tree-status.error {
  color: var(--error);
  font-style: normal;
}

.tree-content {
  flex: 1;
  overflow-y: auto;
  padding-bottom: 0.5rem;
}

.tree-row {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.2rem 0.5rem;
  cursor: pointer;
  font-size: 0.8rem;
  color: var(--text-secondary);
  user-select: none;
}

.tree-row:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.tree-children {
  padding-left: 0.75rem;
}

.arrow {
  font-size: 0.65rem;
  color: var(--text-muted);
  width: 12px;
  text-align: center;
  transition: transform 0.15s;
  flex-shrink: 0;
}

.arrow.expanded {
  transform: rotate(90deg);
}

.icon {
  font-size: 0.8rem;
  width: 14px;
  text-align: center;
  flex-shrink: 0;
}

.db-row .node-name {
  font-weight: 600;
}

.group-row {
  color: var(--text-muted);
  font-size: 0.75rem;
}

.group-name {
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.leaf-row {
  padding-left: 0.75rem;
}

.node-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.row-count {
  font-size: 0.75rem;
  color: var(--text-muted);
  flex-shrink: 0;
}

.badge {
  font-size: 0.7rem;
  color: var(--text-muted);
  background: var(--bg-input);
  padding: 0 0.3rem;
  border-radius: 2px;
  flex-shrink: 0;
}

.loading-dot {
  color: var(--text-muted);
  font-size: 0.8rem;
}

.tree-empty {
  padding: 0.2rem 0.75rem;
  font-size: 0.75rem;
  color: var(--text-muted);
  font-style: italic;
}

.leaf-row.selected {
  background: var(--bg-active);
  color: var(--text-primary);
}

.query-btn {
  font-size: 0.65rem;
  color: var(--text-muted);
  cursor: pointer;
  padding: 1px 3px;
  border-radius: 2px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
}

.leaf-row:hover .query-btn {
  opacity: 1;
}

.query-btn:hover {
  color: var(--success);
  background: var(--bg-input);
}

.add-user-row {
  color: var(--text-muted);
}

.add-user-row:hover {
  color: var(--accent);
}

.add-label {
  font-style: italic;
}
</style>
