<script lang="ts" setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { EditorView, keymap, placeholder } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { MySQL, sql } from '@codemirror/lang-sql'
import { oneDark } from '@codemirror/theme-one-dark'
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
import { bracketMatching } from '@codemirror/language'
import { autocompletion } from '@codemirror/autocomplete'

const props = defineProps<{
  tabId: string
  schema?: Record<string, string[]> | null
}>()

const emit = defineEmits<{
  (e: 'execute', sql: string): void
  (e: 'explain', sql: string): void
  (e: 'cancel'): void
  (e: 'import-csv'): void
  (e: 'import-sql'): void
}>()

const editorContainer = ref<HTMLElement | null>(null)
let view: EditorView | null = null
const running = ref(false)

const sqlCompartment = new Compartment()

const mybenchTheme = EditorView.theme({
  '&': {
    height: '100%',
    fontSize: '14px',
  },
  '.cm-content': {
    fontFamily: "'SF Mono', 'Fira Code', 'Consolas', monospace",
    caretColor: '#c0caf5',
  },
  '.cm-cursor': {
    borderLeftColor: '#c0caf5',
  },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
    backgroundColor: '#33354a !important',
  },
  '.cm-gutters': {
    backgroundColor: '#16161e',
    color: '#565a7e',
    border: 'none',
  },
  '.cm-activeLineGutter': {
    backgroundColor: '#1f2029',
  },
  '.cm-activeLine': {
    backgroundColor: '#1f202940',
  },
  '.cm-tooltip-autocomplete': {
    backgroundColor: '#1f2029 !important',
    border: '1px solid #2e2f3e !important',
  },
  '.cm-tooltip-autocomplete > ul > li': {
    color: '#c0caf5',
  },
  '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
    backgroundColor: '#33354a !important',
    color: '#c0caf5',
  },
}, { dark: true })

const runQueryKeymap = keymap.of([
  {
    key: 'Ctrl-Enter',
    run: () => {
      executeQuery()
      return true
    },
  },
  {
    key: 'Ctrl-Shift-Enter',
    run: () => {
      explainQuery()
      return true
    },
  },
])

function buildSqlExtension(schema?: Record<string, string[]> | null) {
  const opts: any = { dialect: MySQL, upperCaseKeywords: true }
  if (schema && Object.keys(schema).length > 0) {
    opts.schema = schema
  }
  return sql(opts)
}

onMounted(() => {
  if (!editorContainer.value) return

  const state = EditorState.create({
    doc: '',
    extensions: [
      history(),
      bracketMatching(),
      highlightSelectionMatches(),
      sqlCompartment.of(buildSqlExtension(props.schema)),
      autocompletion(),
      oneDark,
      mybenchTheme,
      runQueryKeymap,
      keymap.of([
        ...defaultKeymap,
        ...historyKeymap,
        ...searchKeymap,
      ]),
      placeholder('Enter SQL query...'),
      EditorView.lineWrapping,
    ],
  })

  view = new EditorView({
    state,
    parent: editorContainer.value,
  })
})

onUnmounted(() => {
  view?.destroy()
})

// When schema changes, reconfigure the SQL extension with new completion data
watch(() => props.schema, (newSchema) => {
  if (!view) return
  view.dispatch({
    effects: sqlCompartment.reconfigure(buildSqlExtension(newSchema)),
  })
})

function getSQL(): string {
  if (!view) return ''
  const state = view.state
  const selection = state.sliceDoc(
    state.selection.main.from,
    state.selection.main.to
  )
  if (selection.trim()) return selection.trim()
  return state.doc.toString().trim()
}

function executeQuery() {
  const s = getSQL()
  if (!s) return
  emit('execute', s)
}

function explainQuery() {
  const s = getSQL()
  if (!s) return
  emit('explain', s)
}

function cancelQuery() {
  emit('cancel')
}

function setRunning(val: boolean) {
  running.value = val
}

function focus() {
  view?.focus()
}

function getContent(): string {
  return view?.state.doc.toString() || ''
}

function setContent(content: string) {
  if (!view) return
  view.dispatch({
    changes: { from: 0, to: view.state.doc.length, insert: content },
  })
}

defineExpose({ setRunning, focus, getContent, setContent })
</script>

<template>
  <div class="editor-area">
    <div class="editor-toolbar">
      <button class="primary" @click="executeQuery" :disabled="running" title="Run query (Ctrl+Enter)">
        {{ running ? 'Running...' : 'Run' }}
      </button>
      <button @click="explainQuery" :disabled="running" title="Explain query (Ctrl+Shift+Enter)">Explain</button>
      <button @click="cancelQuery" :disabled="!running" title="Cancel running query">Cancel</button>
      <div class="toolbar-spacer" />
      <button class="import-btn" @click="emit('import-csv')" :disabled="!tabId" title="Import CSV file">Import CSV</button>
      <button class="import-btn" @click="emit('import-sql')" :disabled="!tabId" title="Import/run SQL file">Import SQL</button>
      <span v-if="tabId" class="tab-indicator">{{ tabId }}</span>
    </div>
    <div class="editor-wrapper" ref="editorContainer" />
  </div>
</template>

<style scoped>
.editor-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 120px;
}

.editor-toolbar {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.5rem;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.toolbar-spacer {
  flex: 1;
}

.import-btn {
  padding: 0.2rem 0.5rem;
  font-size: 0.7rem;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  border-radius: 3px;
  cursor: pointer;
}

.import-btn:hover {
  color: var(--accent);
  border-color: var(--accent-dim);
}

.import-btn:disabled {
  opacity: 0.4;
  cursor: default;
}

.tab-indicator {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.editor-wrapper {
  flex: 1;
  overflow: hidden;
}

.editor-wrapper :deep(.cm-editor) {
  height: 100%;
}

.editor-wrapper :deep(.cm-scroller) {
  overflow: auto;
}
</style>
