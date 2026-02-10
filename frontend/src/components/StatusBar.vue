<script lang="ts" setup>
defineProps<{
  connected?: boolean
}>()
</script>

<template>
  <div class="status-bar">
    <span class="status-item" :class="{ active: connected }">
      {{ connected ? 'Connected' : 'Disconnected' }}
    </span>
    <span class="status-spacer" />
    <span class="shortcut-hints">
      <span class="hint">Ctrl+Enter <span class="hint-label">Run</span></span>
      <span class="hint">Ctrl+Shift+Enter <span class="hint-label">Explain</span></span>
      <span class="hint">Ctrl+L <span class="hint-label">Focus editor</span></span>
    </span>
  </div>
</template>

<style scoped>
.status-bar {
  display: flex;
  align-items: center;
  padding: 0 0.75rem;
  height: 24px;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  flex-shrink: 0;
}

.status-item {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.status-item.active {
  color: var(--success);
}

.status-spacer {
  flex: 1;
}

.shortcut-hints {
  display: flex;
  gap: 0.75rem;
}

.hint {
  font-size: 0.7rem;
  color: var(--text-muted);
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  opacity: 0.7;
}

.hint-label {
  font-family: inherit;
  color: var(--text-muted);
  opacity: 0.6;
}
</style>
