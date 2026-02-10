<script lang="ts" setup>
import { ref } from 'vue'

const tabs = ref([
  { id: 1, name: 'Query 1', active: true }
])

function addTab() {
  const id = tabs.value.length + 1
  tabs.value.forEach(t => t.active = false)
  tabs.value.push({ id, name: `Query ${id}`, active: true })
}

function selectTab(id: number) {
  tabs.value.forEach(t => t.active = t.id === id)
}

function closeTab(id: number) {
  if (tabs.value.length <= 1) return
  const idx = tabs.value.findIndex(t => t.id === id)
  const wasActive = tabs.value[idx].active
  tabs.value.splice(idx, 1)
  if (wasActive && tabs.value.length > 0) {
    tabs.value[Math.min(idx, tabs.value.length - 1)].active = true
  }
}
</script>

<template>
  <div class="tab-bar">
    <div class="tabs">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="tab"
        :class="{ active: tab.active }"
        @click="selectTab(tab.id)"
      >
        <span class="tab-name">{{ tab.name }}</span>
        <span
          class="tab-close"
          @click.stop="closeTab(tab.id)"
          v-if="tabs.length > 1"
        >&times;</span>
      </div>
    </div>
    <button class="new-tab" @click="addTab" title="New tab">+</button>
  </div>
</template>

<style scoped>
.tab-bar {
  display: flex;
  align-items: stretch;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  height: 34px;
  flex-shrink: 0;
}

.tabs {
  display: flex;
  overflow-x: auto;
  flex: 1;
}

.tab {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0 0.85rem;
  cursor: pointer;
  color: var(--text-muted);
  border-right: 1px solid var(--border);
  white-space: nowrap;
  font-size: 0.8rem;
  user-select: none;
}

.tab:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
}

.tab.active {
  color: var(--text-primary);
  background: var(--bg-primary);
  border-bottom: 2px solid var(--accent);
}

.tab-close {
  font-size: 1rem;
  line-height: 1;
  color: var(--text-muted);
  padding: 0 2px;
  border-radius: 2px;
}

.tab-close:hover {
  color: var(--error);
  background: var(--bg-active);
}

.new-tab {
  border: none;
  border-radius: 0;
  background: transparent;
  color: var(--text-muted);
  padding: 0 0.75rem;
  font-size: 1rem;
}

.new-tab:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
</style>
