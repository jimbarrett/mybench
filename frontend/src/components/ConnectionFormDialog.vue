<script lang="ts" setup>
import { ref, watch } from 'vue'
import { main } from '../wailsjs/go/models'
import { TestConnection } from '../wailsjs/go/main/App'

const props = defineProps<{
  connection?: main.ConnectionProfile | null
}>()

const emit = defineEmits<{
  (e: 'save', conn: main.ConnectionProfile): void
  (e: 'cancel'): void
}>()

const form = ref<main.ConnectionProfile>(new main.ConnectionProfile({
  id: '',
  name: '',
  host: '127.0.0.1',
  port: 3306,
  username: 'root',
  password: '',
  defaultDb: '',
  useSsl: false,
  sshEnabled: false,
  sshHost: '',
  sshPort: 22,
  sshUser: '',
  sshAuth: 'key',
  sshKeyPath: '',
  sshPassword: '',
  sortOrder: 0,
}))

const testing = ref(false)
const testResult = ref<{ ok: boolean; message: string } | null>(null)
const error = ref('')

watch(() => props.connection, (conn) => {
  if (conn) {
    form.value = new main.ConnectionProfile({ ...conn })
  }
}, { immediate: true })

function validate(): string {
  if (!form.value.name.trim()) return 'Connection name is required'
  if (!form.value.host.trim()) return 'Host is required'
  if (!form.value.port || form.value.port < 1) return 'Valid port is required'
  if (!form.value.username.trim()) return 'Username is required'
  return ''
}

async function testConn() {
  const err = validate()
  if (err) {
    error.value = err
    return
  }

  testing.value = true
  testResult.value = null
  error.value = ''

  try {
    await TestConnection(form.value)
    testResult.value = { ok: true, message: 'Connection successful' }
  } catch (e: any) {
    testResult.value = { ok: false, message: e?.message || String(e) }
  } finally {
    testing.value = false
  }
}

function save() {
  const err = validate()
  if (err) {
    error.value = err
    return
  }
  error.value = ''
  emit('save', form.value)
}
</script>

<template>
  <div class="overlay">
    <div class="dialog">
      <h2 class="dialog-title">
        {{ form.id ? 'Edit Connection' : 'New Connection' }}
      </h2>

      <form @submit.prevent="save">
        <div class="form-grid">
          <div class="field full">
            <label class="field-label">Name</label>
            <input v-model="form.name" class="field-input" placeholder="My Database" autofocus />
          </div>

          <div class="field">
            <label class="field-label">Host</label>
            <input v-model="form.host" class="field-input" placeholder="127.0.0.1" />
          </div>

          <div class="field narrow">
            <label class="field-label">Port</label>
            <input v-model.number="form.port" type="number" class="field-input" />
          </div>

          <div class="field">
            <label class="field-label">Username</label>
            <input v-model="form.username" class="field-input" placeholder="root" />
          </div>

          <div class="field">
            <label class="field-label">Password</label>
            <input v-model="form.password" type="password" class="field-input" />
          </div>

          <div class="field">
            <label class="field-label">Default Database</label>
            <input v-model="form.defaultDb" class="field-input" placeholder="(optional)" />
          </div>

          <div class="field check-field">
            <label class="check-label">
              <input v-model="form.useSsl" type="checkbox" />
              Use SSL/TLS
            </label>
          </div>
        </div>

        <!-- SSH Section -->
        <div class="section-divider">
          <label class="check-label">
            <input v-model="form.sshEnabled" type="checkbox" />
            Connect via SSH tunnel
          </label>
        </div>

        <div v-if="form.sshEnabled" class="form-grid">
          <div class="field">
            <label class="field-label">SSH Host</label>
            <input v-model="form.sshHost" class="field-input" />
          </div>

          <div class="field narrow">
            <label class="field-label">SSH Port</label>
            <input v-model.number="form.sshPort" type="number" class="field-input" />
          </div>

          <div class="field">
            <label class="field-label">SSH User</label>
            <input v-model="form.sshUser" class="field-input" />
          </div>

          <div class="field">
            <label class="field-label">Auth Method</label>
            <select v-model="form.sshAuth" class="field-input">
              <option value="key">SSH Key</option>
              <option value="password">Password</option>
            </select>
          </div>

          <div v-if="form.sshAuth === 'key'" class="field full">
            <label class="field-label">Key Path</label>
            <input v-model="form.sshKeyPath" class="field-input" placeholder="~/.ssh/id_rsa" />
          </div>

          <div v-if="form.sshAuth === 'password'" class="field full">
            <label class="field-label">SSH Password</label>
            <input v-model="form.sshPassword" type="password" class="field-input" />
          </div>
        </div>

        <div v-if="error" class="error">{{ error }}</div>

        <div v-if="testResult" class="test-result" :class="{ success: testResult.ok, fail: !testResult.ok }">
          {{ testResult.message }}
        </div>

        <div class="dialog-actions">
          <button type="button" @click="emit('cancel')">Cancel</button>
          <button type="button" @click="testConn" :disabled="testing">
            {{ testing ? 'Testing...' : 'Test' }}
          </button>
          <button type="submit" class="primary">Save</button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
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
  padding: 1.5rem;
  width: 480px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.dialog-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 1rem;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.6rem;
}

.field {
  display: flex;
  flex-direction: column;
}

.field.full {
  grid-column: 1 / -1;
}

.field.narrow {
  max-width: 120px;
}

.field-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 0.25rem;
}

.field-input {
  width: 100%;
}

.check-field {
  justify-content: flex-end;
  grid-column: 1 / -1;
}

.check-label {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
  cursor: pointer;
}

.check-label input[type="checkbox"] {
  accent-color: var(--accent);
}

.section-divider {
  margin: 0.75rem 0;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border);
}

.error {
  font-size: 0.8rem;
  color: var(--error);
  margin-top: 0.75rem;
}

.test-result {
  font-size: 0.8rem;
  margin-top: 0.75rem;
  padding: 0.4rem 0.6rem;
  border-radius: 4px;
}

.test-result.success {
  color: var(--success);
  background: rgba(158, 206, 106, 0.1);
}

.test-result.fail {
  color: var(--error);
  background: rgba(247, 118, 142, 0.1);
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1rem;
}
</style>
