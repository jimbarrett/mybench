<script lang="ts" setup>
import { ref } from 'vue'
import { CreateUser } from '../wailsjs/go/main/App'

const props = defineProps<{
  tabId: string
}>()

const emit = defineEmits<{
  (e: 'created'): void
  (e: 'cancel'): void
}>()

const username = ref('')
const host = ref('%')
const password = ref('')
const plugin = ref('caching_sha2_password')
const error = ref('')
const saving = ref(false)

const plugins = [
  'caching_sha2_password',
  'mysql_native_password',
]

async function handleSubmit() {
  error.value = ''
  if (!username.value.trim()) {
    error.value = 'Username is required'
    return
  }
  if (!password.value) {
    error.value = 'Password is required'
    return
  }

  saving.value = true
  try {
    await CreateUser(props.tabId, username.value.trim(), host.value, password.value, plugin.value)
    emit('created')
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="dialog-overlay" @click.self="emit('cancel')">
    <div class="dialog">
      <h2 class="dialog-title">Create User</h2>

      <div class="form-group">
        <label>Username</label>
        <input
          v-model="username"
          type="text"
          placeholder="username"
          autofocus
          @keydown.enter="handleSubmit"
        />
      </div>

      <div class="form-group">
        <label>Host</label>
        <input
          v-model="host"
          type="text"
          placeholder="% (any host)"
        />
        <span class="hint">Use % for any host, or specify an IP/hostname</span>
      </div>

      <div class="form-group">
        <label>Password</label>
        <input
          v-model="password"
          type="password"
          placeholder="password"
          @keydown.enter="handleSubmit"
        />
      </div>

      <div class="form-group">
        <label>Authentication Plugin</label>
        <select v-model="plugin">
          <option v-for="p in plugins" :key="p" :value="p">{{ p }}</option>
        </select>
      </div>

      <div v-if="error" class="error-msg">{{ error }}</div>

      <div class="dialog-actions">
        <button @click="emit('cancel')">Cancel</button>
        <button class="primary" @click="handleSubmit" :disabled="saving">
          {{ saving ? 'Creating...' : 'Create User' }}
        </button>
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

.dialog {
  background: var(--bg-sidebar);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1.25rem;
  width: 380px;
  max-width: 90vw;
}

.dialog-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 1rem;
}

.form-group {
  margin-bottom: 0.75rem;
}

.form-group label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  margin-bottom: 0.25rem;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 0.4rem 0.6rem;
  font-size: 0.85rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  box-sizing: border-box;
}

.form-group select {
  cursor: pointer;
}

.form-group input:focus,
.form-group select:focus {
  border-color: var(--accent);
  outline: none;
}

.hint {
  font-size: 0.7rem;
  color: var(--text-muted);
  margin-top: 0.15rem;
  display: block;
}

.error-msg {
  color: var(--error);
  font-size: 0.8rem;
  margin-bottom: 0.75rem;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1rem;
}
</style>
