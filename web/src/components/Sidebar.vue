<script setup lang="ts">
import { ref } from 'vue'
import { useSessionStore } from '../stores/session'
import { formatTime } from '../utils/format'
import WorkspaceSelector from './WorkspaceSelector.vue'
import { useI18n } from '../composables/useI18n'

const emit = defineEmits<{ collapse: [] }>()

const store = useSessionStore()
const { filteredSessions, currentSessionId } = store
const { t } = useI18n()

const deleteModalOpen = ref(false)
const deleteTargetId = ref<string | null>(null)

function handleDelete(e: Event, id: string) {
  e.stopPropagation()
  deleteTargetId.value = id
  deleteModalOpen.value = true
}

function confirmDelete() {
  if (deleteTargetId.value) {
    store.removeSession(deleteTargetId.value)
  }
  closeModal()
}

function closeModal() {
  deleteModalOpen.value = false
  deleteTargetId.value = null
}
</script>

<template>
  <aside class="sidebar">
    <WorkspaceSelector @collapse="emit('collapse')" />
    <div class="sidebar-header">
      <h2>Chats</h2>
      <button class="new-btn" :title="t('sidebar.new_chat')" @click="store.createNewSession">
        +
      </button>
    </div>
    <div class="session-list">
      <div
        v-for="session in filteredSessions"
        :key="session.id"
        class="session-item"
        :class="{ active: session.id === currentSessionId }"
        @click="store.selectSession(session.id)"
      >
        <div class="session-title">{{ session.title }}</div>
        <div class="session-meta">
          <span class="session-agent">{{ session.activeAgent }}</span>
          <span class="session-time">{{ formatTime(session.updatedAt) }}</span>
        </div>
        <button
          class="session-delete"
          title="Delete"
          @click="(e) => handleDelete(e, session.id)"
        >
          &times;
        </button>
      </div>
      <div v-if="filteredSessions.length === 0" class="no-sessions">
        No chats in this workspace
      </div>
    </div>

    <!-- Delete Confirm Modal -->
    <Teleport to="body">
      <div v-if="deleteModalOpen" class="modal-overlay" @click="closeModal">
        <div class="modal-content" @click.stop>
          <div class="modal-title">Delete Chat</div>
          <div class="modal-body">Are you sure you want to delete this chat?</div>
          <div class="modal-actions">
            <button class="btn-cancel" @click="closeModal">Cancel</button>
            <button class="btn-delete" @click="confirmDelete">Delete</button>
          </div>
        </div>
      </div>
    </Teleport>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 260px;
  background: var(--bg-surface);
  border-right: 1px solid var(--bg-element);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  transition: width var(--duration-normal) var(--ease-snappy);
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 16px 12px;
}

.sidebar-header h2 {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  margin: 0;
}

.new-btn {
  width: 24px;
  height: 24px;
  border: 1px solid var(--bg-element);
  background: transparent;
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--duration-fast);
}

.new-btn:hover {
  background: var(--bg-surface-hover);
  color: var(--text-primary);
  border-color: var(--text-tertiary);
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  /* Visual scrollbar hiding but functional */
  scrollbar-width: none;
}

.session-list::-webkit-scrollbar {
  display: none;
}

.session-item {
  padding: 10px 12px;
  border-radius: var(--radius-md);
  cursor: pointer;
  margin-bottom: 2px;
  position: relative;
  transition: background var(--duration-fast), transform var(--duration-fast);
  border: 1px solid transparent;
}

.session-item:hover {
  background: var(--bg-surface-hover);
}

.session-item.active {
  background: var(--bg-element);
}

.session-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  height: 12px;
  width: 2px;
  background-color: var(--accent-primary);
  border-radius: 0 2px 2px 0;
}

.session-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 2px;
  transition: color var(--duration-fast);
}

.session-item.active .session-title,
.session-item:hover .session-title {
  color: var(--text-primary);
}

.session-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 10px;
  color: var(--text-tertiary);
}

.session-agent {
  font-family: var(--font-mono);
  font-size: 9px;
  opacity: 0.7;
}

.session-time {
  margin-left: auto;
}

.session-delete {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 20px;
  height: 20px;
  border: none;
  background: var(--bg-root);
  color: var(--text-tertiary);
  font-size: 14px;
  cursor: pointer;
  border-radius: var(--radius-sm);
  opacity: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--duration-fast);
}

.session-item:hover .session-delete {
  opacity: 1;
}

.session-delete:hover {
  background: var(--accent-error);
  color: #fff;
}

.no-sessions {
  text-align: center;
  color: var(--text-tertiary);
  padding: 40px 20px;
  font-size: 12px;
}
</style>

<style>
/* Modal styles (global for Teleport) */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.8);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.modal-content {
  background: var(--bg-element);
  border: 1px solid var(--accent-subtle);
  border-radius: var(--radius-lg);
  padding: 24px;
  min-width: 320px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
}

.modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.modal-body {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 24px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.modal-actions button {
  padding: 8px 16px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--duration-fast);
}

.btn-cancel {
  background: transparent;
  border: 1px solid var(--accent-subtle);
  color: var(--text-secondary);
}

.btn-cancel:hover {
  border-color: var(--text-primary);
  color: var(--text-primary);
}

.btn-delete {
  background: var(--accent-error);
  border: 1px solid var(--accent-error);
  color: white;
}

.btn-delete:hover {
  opacity: 0.9;
}
</style>
