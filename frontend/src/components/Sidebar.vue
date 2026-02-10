<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { main } from '../wailsjs/go/models'
import {
  ListConnections,
  SaveConnection,
  DeleteConnection,
  Connect,
  Disconnect,
} from '../wailsjs/go/main/App'
import ConnectionFormDialog from './ConnectionFormDialog.vue'
import SchemaTree from './SchemaTree.vue'

const connections = ref<main.ConnectionProfile[]>([])
const activeConns = ref<Record<string, boolean>>({})
const showForm = ref(false)
const editingConnection = ref<main.ConnectionProfile | null>(null)
const selectedProfileId = ref<string>('')

const emit = defineEmits<{
  (e: 'connected', profileId: string, tabId: string): void
  (e: 'disconnected', profileId: string): void
  (e: 'select-table', tabId: string, db: string, table: string): void
  (e: 'query-table', tabId: string, db: string, table: string): void
  (e: 'select-user', tabId: string, user: string, host: string): void
  (e: 'create-user', tabId: string): void
}>()

onMounted(async () => {
  await loadConnections()
})

async function loadConnections() {
  try {
    connections.value = await ListConnections()
  } catch (e) {
    console.error('Failed to load connections:', e)
  }
}

function openNewForm() {
  editingConnection.value = null
  showForm.value = true
}

function openEditForm(conn: main.ConnectionProfile) {
  editingConnection.value = conn
  showForm.value = true
}

async function handleSave(conn: main.ConnectionProfile) {
  try {
    await SaveConnection(conn)
    showForm.value = false
    editingConnection.value = null
    await loadConnections()
  } catch (e: any) {
    console.error('Failed to save connection:', e)
  }
}

async function handleDelete(id: string) {
  try {
    await DeleteConnection(id)
    if (activeConns.value[id]) {
      const tabId = 'tab-' + id
      await Disconnect(tabId)
      activeConns.value[id] = false
      emit('disconnected', id)
    }
    await loadConnections()
  } catch (e) {
    console.error('Failed to delete connection:', e)
  }
}

async function handleConnect(profile: main.ConnectionProfile) {
  const tabId = 'tab-' + profile.id
  try {
    await Connect(tabId, profile.id)
    activeConns.value[profile.id] = true
    selectedProfileId.value = profile.id
    emit('connected', profile.id, tabId)
  } catch (e: any) {
    console.error('Connection failed:', e)
    activeConns.value[profile.id] = false
  }
}

async function handleDisconnect(profileId: string) {
  const tabId = 'tab-' + profileId
  try {
    await Disconnect(tabId)
  } catch (e) {
    console.error('Disconnect failed:', e)
  }
  activeConns.value[profileId] = false
  emit('disconnected', profileId)
}

function connStatus(profileId: string): string {
  return activeConns.value[profileId] ? 'connected' : 'disconnected'
}

function activeTabId(): string {
  if (!selectedProfileId.value) return ''
  return 'tab-' + selectedProfileId.value
}

function onSelectTable(db: string, table: string) {
  emit('select-table', activeTabId(), db, table)
}

function onQueryTable(db: string, table: string) {
  emit('query-table', activeTabId(), db, table)
}

function onSelectUser(user: string, host: string) {
  emit('select-user', activeTabId(), user, host)
}

function onCreateUser() {
  emit('create-user', activeTabId())
}

const schemaTreeRef = ref<InstanceType<typeof SchemaTree> | null>(null)

function refreshUsers() {
  schemaTreeRef.value?.refreshUsers()
}

defineExpose({ loadConnections, refreshUsers })
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <h1 class="app-title">mybench</h1>
      <span class="version">v0.1.0</span>
    </div>

    <div class="sidebar-section">
      <div class="section-label">Connections</div>
      <div class="connection-list">
        <div
          v-for="conn in connections"
          :key="conn.id"
          class="connection-item"
          :class="{ selected: conn.id === selectedProfileId }"
        >
          <span class="connection-status" :class="connStatus(conn.id)" />
          <span class="connection-name" @click="selectedProfileId = conn.id">{{ conn.name }}</span>
          <span class="connection-actions">
            <button
              v-if="!activeConns[conn.id]"
              class="icon-btn"
              title="Connect"
              @click="handleConnect(conn)"
            >&#9654;</button>
            <button
              v-else
              class="icon-btn disconnect"
              title="Disconnect"
              @click="handleDisconnect(conn.id)"
            >&#9632;</button>
            <button
              class="icon-btn"
              title="Edit"
              @click="openEditForm(conn)"
            >&#9998;</button>
            <button
              class="icon-btn delete"
              title="Delete"
              @click="handleDelete(conn.id)"
            >&times;</button>
          </span>
        </div>

        <div v-if="connections.length === 0" class="empty-state">
          No saved connections
        </div>

        <button class="add-connection" @click="openNewForm">+ New Connection</button>
      </div>
    </div>

    <SchemaTree
      ref="schemaTreeRef"
      :tabId="activeTabId()"
      :connected="!!selectedProfileId && !!activeConns[selectedProfileId]"
      @select-table="onSelectTable"
      @query-table="onQueryTable"
      @select-user="onSelectUser"
      @create-user="onCreateUser"
    />

    <ConnectionFormDialog
      v-if="showForm"
      :connection="editingConnection"
      @save="handleSave"
      @cancel="showForm = false"
    />
  </aside>
</template>

<style scoped>
.sidebar {
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
}

.app-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--accent);
}

.version {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.sidebar-section {
  padding: 0.5rem 0;
}

.section-label {
  padding: 0.25rem 1rem;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
}

.connection-list {
  padding: 0.25rem 0;
}

.connection-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.35rem 0.5rem 0.35rem 1rem;
  color: var(--text-secondary);
}

.connection-item:hover {
  background: var(--bg-hover);
}

.connection-item.selected {
  background: var(--bg-active);
  color: var(--text-primary);
}

.connection-status {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-muted);
  flex-shrink: 0;
}

.connection-status.connected {
  background: var(--success);
}

.connection-name {
  flex: 1;
  font-size: 0.85rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.connection-actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s;
}

.connection-item:hover .connection-actions {
  opacity: 1;
}

.icon-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 2px 4px;
  font-size: 0.75rem;
  border-radius: 2px;
  line-height: 1;
}

.icon-btn:hover {
  color: var(--accent);
  background: var(--bg-input);
}

.icon-btn.disconnect:hover {
  color: var(--warning);
}

.icon-btn.delete:hover {
  color: var(--error);
}

.add-connection {
  width: calc(100% - 1rem);
  margin: 0.35rem 0.5rem;
  padding: 0.3rem;
  font-size: 0.8rem;
  text-align: center;
  border-style: dashed;
  color: var(--text-muted);
}

.add-connection:hover {
  color: var(--accent);
  border-color: var(--accent-dim);
}

.empty-state {
  padding: 1rem;
  font-size: 0.8rem;
  color: var(--text-muted);
  text-align: center;
  font-style: italic;
}
</style>
