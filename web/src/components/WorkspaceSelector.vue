<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useSessionStore } from '../stores/session'

const emit = defineEmits<{ collapse: [] }>()

const store = useSessionStore()
const { workspaces, currentWorkspace, currentWorkspaceInfo } = store

const showAddForm = ref(false)
const isOpen = ref(false)
const newName = ref('')
const newPath = ref('')
const error = ref('')

const currentWorkspaceName = computed(() => {
  const ws = workspaces.value.find(w => w.id === currentWorkspace.value)
  return ws ? ws.name : 'Select Workspace'
})

function toggleDropdown() {
  isOpen.value = !isOpen.value
}

function selectWorkspace(id: string) {
  store.setWorkspace(id)
  isOpen.value = false
}

// Close dropdown when clicking outside
function handleGlobalClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (isOpen.value && !target.closest('.workspace-selector')) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleGlobalClick)
})

onUnmounted(() => {
  document.removeEventListener('click', handleGlobalClick)
})

function openAddForm() {
  showAddForm.value = true
  newName.value = ''
  newPath.value = ''
  error.value = ''
  isOpen.value = false
}

function closeAddForm() {
  showAddForm.value = false
}

async function handleAdd() {
  if (!newName.value.trim() || !newPath.value.trim()) {
    error.value = 'Name and path are required'
    return
  }

  const result = await store.addWorkspace(newName.value.trim(), newPath.value.trim())
  if (result) {
    error.value = result
  } else {
    closeAddForm()
  }
}
</script>

<template>
  <div class="workspace-selector">
    <div class="selector-header">
      <div class="label">WORKSPACE</div>
      <div class="header-actions">
        <button class="add-btn" title="Add Workspace" @click.stop="openAddForm">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
        </button>
        <button class="collapse-btn" title="Hide sidebar" @click.stop="emit('collapse')">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="9" y1="3" x2="9" y2="21"/><polyline points="14 9 11 12 14 15"/></svg>
        </button>
      </div>
    </div>

    <div class="dropdown-trigger" @click.stop="toggleDropdown" :class="{ active: isOpen }">
      <div class="current-value">
        <span class="ws-name">{{ currentWorkspaceName }}</span>
        <span class="trigger-icon">▼</span>
      </div>
      <div v-if="currentWorkspaceInfo" class="ws-path">
        {{ currentWorkspaceInfo.path }}
      </div>
    </div>

    <div v-if="isOpen" class="dropdown-menu">
      <div
        v-for="ws in workspaces"
        :key="ws.id"
        class="dropdown-item"
        :class="{ selected: ws.id === currentWorkspace }"
        @click.stop="selectWorkspace(ws.id)"
      >
        <div class="item-name">{{ ws.name }}</div>
        <div class="item-path">{{ ws.path }}</div>
      </div>
      <div v-if="workspaces.length === 0" class="dropdown-item empty">
        No workspaces
      </div>
    </div>

    <!-- Add Workspace Modal -->
    <Teleport to="body">
      <div v-if="showAddForm" class="modal-overlay" @click.self="closeAddForm">
        <div class="modal-content">
          <div class="modal-title">Add Workspace</div>

          <div class="form-group">
            <label>Name</label>
            <input v-model="newName" type="text" placeholder="My Project" class="modal-input" />
          </div>

          <div class="form-group">
            <label>Path</label>
            <input v-model="newPath" type="text" placeholder="/path/to/project" class="modal-input" />
          </div>

          <div v-if="error" class="error-msg">{{ error }}</div>

          <div class="modal-actions">
            <button class="btn-cancel" @click="closeAddForm">Cancel</button>
            <button class="btn-add" @click="handleAdd">Add</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.workspace-selector {
  position: relative;
  padding: 16px 14px 8px; /* Standardize padding */
  border-bottom: 1px solid var(--bg-element);
  z-index: 20; /* Ensure dropdown displays above session list */
}

.selector-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.label {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.1em;
  color: var(--text-tertiary);
}

.add-btn {
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--radius-sm);
  display: flex;
  transition: all var(--duration-fast);
}

.add-btn:hover {
  background: var(--bg-element);
  color: var(--text-primary);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.collapse-btn {
  background: transparent;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--radius-sm);
  display: flex;
  font-size: 12px;
  transition: all var(--duration-fast);
}

.collapse-btn:hover {
  background: var(--bg-element);
  color: var(--text-primary);
}

/* Custom Dropdown Trigger */
.dropdown-trigger {
  background: var(--bg-root);
  border: 1px solid var(--bg-element);
  border-radius: var(--radius-md);
  padding: 8px 10px;
  cursor: pointer;
  transition: all var(--duration-fast);
}

.dropdown-trigger:hover,
.dropdown-trigger.active {
  border-color: var(--text-secondary);
  background: var(--bg-surface-hover);
}

.current-value {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.ws-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.trigger-icon {
  font-size: 8px;
  color: var(--text-tertiary);
  transition: transform var(--duration-fast);
}

.dropdown-trigger.active .trigger-icon {
  transform: rotate(180deg);
}

.ws-path {
  margin-top: 4px;
  font-size: 10px;
  color: var(--text-tertiary);
  font-family: var(--font-mono);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Dropdown Menu */
.dropdown-menu {
  position: absolute;
  top: 100%;
  left: 12px;
  right: 12px;
  margin-top: 6px;
  background: var(--bg-surface);
  border: 1px solid var(--bg-element);
  border-radius: var(--radius-md);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5); /* Deep shadow for void feel */
  padding: 4px;
  z-index: 100;
  max-height: 300px;
  overflow-y: auto;
}

[data-theme="light"] .dropdown-menu {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.dropdown-item {
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--duration-fast);
  margin-bottom: 2px;
}

.dropdown-item:hover {
  background: var(--bg-element);
}

.dropdown-item.selected {
  background: var(--bg-surface-hover);
  border-left: 2px solid var(--text-primary);
}

.item-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.item-path {
  font-size: 10px;
  color: var(--text-tertiary);
  font-family: var(--font-mono);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dropdown-item.empty {
  color: var(--text-tertiary);
  font-style: italic;
  text-align: center;
  padding: 12px;
}

/* Modal Styles */
.modal-input {
  width: 100%;
  background: var(--bg-root);
  border: 1px solid var(--bg-element);
  color: var(--text-primary);
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  font-size: 13px;
}

.modal-input:focus {
  outline: none;
  border-color: var(--text-secondary);
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 11px;
  color: var(--text-secondary);
  margin-bottom: 6px;
  font-weight: 500;
}

.error-msg {
  color: var(--accent-error);
  font-size: 12px;
  margin-bottom: 16px;
  padding: 8px;
  background: rgba(220, 38, 38, 0.1);
  border-radius: var(--radius-sm);
}

.btn-add {
  background: var(--text-primary);
  color: var(--bg-root);
  border: none;
  font-weight: 600;
}

.btn-add:hover {
  opacity: 0.9;
}
</style>
