<script lang="ts" setup>
import { ref } from 'vue'

const props = defineProps<{
  isNew: boolean
}>()

const emit = defineEmits<{
  (e: 'submit', password: string): void
}>()

const password = ref('')
const confirm = ref('')
const error = ref('')

function submit() {
  error.value = ''
  if (props.isNew) {
    if (password.value.length < 4) {
      error.value = 'Password must be at least 4 characters'
      return
    }
    if (password.value !== confirm.value) {
      error.value = 'Passwords do not match'
      return
    }
  }
  if (!password.value) {
    error.value = 'Password is required'
    return
  }
  emit('submit', password.value)
}
</script>

<template>
  <div class="overlay">
    <div class="dialog">
      <h2 class="dialog-title">
        {{ isNew ? 'Create Master Password' : 'Unlock mybench' }}
      </h2>
      <p class="dialog-desc">
        {{ isNew
          ? 'This password encrypts your saved database credentials. You\'ll enter it each time you open mybench.'
          : 'Enter your master password to unlock saved connections.'
        }}
      </p>

      <form @submit.prevent="submit">
        <div class="field">
          <label class="field-label">Master Password</label>
          <input
            v-model="password"
            type="password"
            class="field-input"
            autofocus
            placeholder="Enter password..."
          />
        </div>

        <div v-if="isNew" class="field">
          <label class="field-label">Confirm Password</label>
          <input
            v-model="confirm"
            type="password"
            class="field-input"
            placeholder="Confirm password..."
          />
        </div>

        <div v-if="error" class="error">{{ error }}</div>

        <div class="dialog-actions">
          <button type="submit" class="primary">
            {{ isNew ? 'Create' : 'Unlock' }}
          </button>
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
  width: 380px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.dialog-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.5rem;
}

.dialog-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-bottom: 1rem;
  line-height: 1.4;
}

.field {
  margin-bottom: 0.75rem;
}

.field-label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 0.3rem;
}

.field-input {
  width: 100%;
}

.error {
  font-size: 0.8rem;
  color: var(--error);
  margin-bottom: 0.75rem;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}
</style>
