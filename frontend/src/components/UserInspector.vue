<script lang="ts" setup>
import { ref, watch } from 'vue'
import { database } from '../wailsjs/go/models'
import {
  GetUserDetail,
  DropUser,
  ChangeUserPassword,
  GrantPrivileges,
  RevokePrivileges,
} from '../wailsjs/go/main/App'

const props = defineProps<{
  tabId: string
  user: string
  host: string
}>()

const emit = defineEmits<{
  (e: 'user-changed'): void
}>()

const detail = ref<database.UserDetail | null>(null)
const loading = ref(false)
const error = ref('')
const activeTab = ref<'grants' | 'password' | 'grant' | 'revoke'>('grants')

// Password change
const newPassword = ref('')
const passwordMsg = ref('')
const passwordErr = ref('')

// Grant form
const grantPrivs = ref('SELECT')
const grantOn = ref('*.*')
const grantMsg = ref('')
const grantErr = ref('')

// Revoke form
const revokePrivs = ref('SELECT')
const revokeOn = ref('*.*')
const revokeMsg = ref('')
const revokeErr = ref('')

watch(
  () => [props.tabId, props.user, props.host],
  async () => {
    if (!props.tabId || !props.user) {
      detail.value = null
      return
    }
    await loadDetail()
  },
  { immediate: true }
)

async function loadDetail() {
  loading.value = true
  error.value = ''
  try {
    detail.value = await GetUserDetail(props.tabId, props.user, props.host)
  } catch (e: any) {
    error.value = e?.message || String(e)
    detail.value = null
  } finally {
    loading.value = false
  }
}

async function handleDropUser() {
  if (!confirm(`Drop user '${props.user}'@'${props.host}'? This cannot be undone.`)) return
  try {
    await DropUser(props.tabId, props.user, props.host)
    emit('user-changed')
  } catch (e: any) {
    error.value = e?.message || String(e)
  }
}

async function handleChangePassword() {
  passwordMsg.value = ''
  passwordErr.value = ''
  if (!newPassword.value) {
    passwordErr.value = 'Password cannot be empty'
    return
  }
  try {
    await ChangeUserPassword(props.tabId, props.user, props.host, newPassword.value)
    passwordMsg.value = 'Password changed successfully'
    newPassword.value = ''
  } catch (e: any) {
    passwordErr.value = e?.message || String(e)
  }
}

async function handleGrant() {
  grantMsg.value = ''
  grantErr.value = ''
  if (!grantPrivs.value) {
    grantErr.value = 'Specify privileges to grant'
    return
  }
  try {
    await GrantPrivileges(props.tabId, props.user, props.host, grantPrivs.value, grantOn.value)
    grantMsg.value = 'Privileges granted'
    await loadDetail()
  } catch (e: any) {
    grantErr.value = e?.message || String(e)
  }
}

async function handleRevoke() {
  revokeMsg.value = ''
  revokeErr.value = ''
  if (!revokePrivs.value) {
    revokeErr.value = 'Specify privileges to revoke'
    return
  }
  try {
    await RevokePrivileges(props.tabId, props.user, props.host, revokePrivs.value, revokeOn.value)
    revokeMsg.value = 'Privileges revoked'
    await loadDetail()
  } catch (e: any) {
    revokeErr.value = e?.message || String(e)
  }
}
</script>

<template>
  <div class="inspector">
    <div class="inspector-header">
      <span class="inspector-title">
        <span class="user-icon">&#9873;</span>
        '{{ user }}'@'{{ host }}'
      </span>
      <span v-if="detail" class="plugin-badge">{{ detail.plugin }}</span>
      <button class="drop-btn" @click="handleDropUser" title="Drop user">Drop</button>
    </div>

    <div class="inspector-tabs">
      <span
        class="inspector-tab"
        :class="{ active: activeTab === 'grants' }"
        @click="activeTab = 'grants'"
      >Grants</span>
      <span
        class="inspector-tab"
        :class="{ active: activeTab === 'password' }"
        @click="activeTab = 'password'"
      >Password</span>
      <span
        class="inspector-tab"
        :class="{ active: activeTab === 'grant' }"
        @click="activeTab = 'grant'"
      >Grant</span>
      <span
        class="inspector-tab"
        :class="{ active: activeTab === 'revoke' }"
        @click="activeTab = 'revoke'"
      >Revoke</span>
    </div>

    <div v-if="loading" class="inspector-status">Loading...</div>
    <div v-else-if="error" class="inspector-status error">{{ error }}</div>

    <div v-else-if="detail" class="inspector-content">
      <!-- Grants -->
      <div v-if="activeTab === 'grants'" class="grants-list">
        <div v-for="(grant, i) in detail.grants" :key="i" class="grant-row mono">
          {{ grant }}
        </div>
        <div v-if="!detail.grants || detail.grants.length === 0" class="empty-state">
          No grants found
        </div>
      </div>

      <!-- Password -->
      <div v-if="activeTab === 'password'" class="form-section">
        <label class="form-label">New Password</label>
        <input
          v-model="newPassword"
          type="password"
          class="form-input"
          placeholder="Enter new password"
          @keydown.enter="handleChangePassword"
        />
        <button class="form-btn" @click="handleChangePassword">Change Password</button>
        <div v-if="passwordMsg" class="form-msg success">{{ passwordMsg }}</div>
        <div v-if="passwordErr" class="form-msg error">{{ passwordErr }}</div>
      </div>

      <!-- Grant -->
      <div v-if="activeTab === 'grant'" class="form-section">
        <label class="form-label">Privileges</label>
        <input
          v-model="grantPrivs"
          class="form-input mono"
          placeholder="e.g. SELECT, INSERT, UPDATE"
        />
        <label class="form-label">On</label>
        <input
          v-model="grantOn"
          class="form-input mono"
          placeholder="e.g. *.* or dbname.*"
        />
        <button class="form-btn" @click="handleGrant">Grant</button>
        <div v-if="grantMsg" class="form-msg success">{{ grantMsg }}</div>
        <div v-if="grantErr" class="form-msg error">{{ grantErr }}</div>
      </div>

      <!-- Revoke -->
      <div v-if="activeTab === 'revoke'" class="form-section">
        <label class="form-label">Privileges</label>
        <input
          v-model="revokePrivs"
          class="form-input mono"
          placeholder="e.g. SELECT, INSERT, UPDATE"
        />
        <label class="form-label">On</label>
        <input
          v-model="revokeOn"
          class="form-input mono"
          placeholder="e.g. *.* or dbname.*"
        />
        <button class="form-btn" @click="handleRevoke">Revoke</button>
        <div v-if="revokeMsg" class="form-msg success">{{ revokeMsg }}</div>
        <div v-if="revokeErr" class="form-msg error">{{ revokeErr }}</div>
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
  gap: 0.5rem;
}

.inspector-title {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
  flex: 1;
}

.user-icon {
  color: var(--accent);
}

.plugin-badge {
  font-size: 0.75rem;
  color: var(--text-muted);
  background: var(--bg-input);
  padding: 1px 6px;
  border-radius: 2px;
}

.drop-btn {
  font-size: 0.7rem;
  padding: 2px 8px;
  color: var(--error);
  border-color: var(--error);
  background: transparent;
}

.drop-btn:hover {
  background: rgba(247, 118, 142, 0.15);
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

.grants-list {
  padding: 0.5rem;
}

.grant-row {
  padding: 0.35rem 0.5rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border);
  line-height: 1.5;
  word-break: break-word;
}

.grant-row:last-child {
  border-bottom: none;
}

.grant-row:hover {
  background: var(--bg-hover);
}

.empty-state {
  padding: 1rem;
  font-size: 0.8rem;
  color: var(--text-muted);
  text-align: center;
  font-style: italic;
}

.form-section {
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-width: 400px;
}

.form-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.form-input {
  padding: 0.4rem 0.6rem;
  font-size: 0.85rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
}

.form-input:focus {
  border-color: var(--accent);
  outline: none;
}

.form-btn {
  align-self: flex-start;
  padding: 0.35rem 0.75rem;
  font-size: 0.8rem;
}

.form-msg {
  font-size: 0.8rem;
}

.form-msg.success {
  color: var(--success);
}

.form-msg.error {
  color: var(--error);
}

.mono {
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
}
</style>
