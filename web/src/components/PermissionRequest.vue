<script setup lang="ts">
import { ref } from 'vue'
import type { PermissionRequest } from '../types'
import { confirmPermission } from '../api'

const props = defineProps<{
  request: PermissionRequest
  agentId: string
}>()

const emit = defineEmits<{
  confirmed: []
}>()

const isConfirming = ref(false)

async function handleOption(optionId: string) {
  isConfirming.value = true
  try {
    await confirmPermission(
      props.agentId,
      props.request.toolCall.toolCallId,
      optionId
    )
    emit('confirmed')
  } catch (err) {
    console.error('Failed to confirm permission:', err)
  } finally {
    isConfirming.value = false
  }
}

function getOptionClass(kind: string): string {
  if (kind.includes('allow')) return 'allow'
  return 'reject'
}

function formatInput(rawInput?: Record<string, unknown>): string {
  if (!rawInput) return ''
  if (rawInput.command) return String(rawInput.command)
  if (rawInput.file_path) return String(rawInput.file_path)
  return JSON.stringify(rawInput, null, 2)
}
</script>

<template>
  <div class="permission-request">
    <div class="permission-header">
      <span class="icon">&#9888;</span>
      <span class="title">Permission Required</span>
    </div>

    <div class="permission-content">
      <div class="tool-info">
        <span class="tool-title">{{ request.toolCall.title || 'Tool Call' }}</span>
        <span v-if="request.toolCall.kind" class="tool-kind">{{ request.toolCall.kind }}</span>
      </div>

      <div v-if="request.toolCall.rawInput" class="tool-input">
        <code>{{ formatInput(request.toolCall.rawInput) }}</code>
      </div>
    </div>

    <div class="permission-options">
      <button v-for="option in request.options" :key="option.optionId"
        :class="['option-btn', getOptionClass(option.kind)]" :disabled="isConfirming"
        @click="handleOption(option.optionId)">
        {{ option.name }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.permission-request {
  background: var(--bg-surface);
  border: 1px solid var(--bg-element);
  border-radius: var(--radius-md);
  padding: 20px;
  margin: 12px 0;
  transition: all var(--duration-normal) var(--ease-snappy);
}

.permission-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.icon {
  font-size: 20px;
  color: #f59e0b;
  display: flex;
  align-items: center;
  justify-content: center;
}

.title {
  font-weight: 600;
  color: var(--text-primary);
  font-family: var(--font-sans);
  font-size: 15px;
}

.permission-content {
  margin-bottom: 20px;
}

.tool-info {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.tool-title {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 14px;
}

.tool-kind {
  background: var(--bg-element);
  padding: 4px 10px;
  border-radius: var(--radius-pill);
  font-size: 12px;
  color: var(--text-secondary);
  border: 1px solid var(--bg-surface-hover);
}

.tool-input {
  background: var(--bg-element);
  padding: 12px 16px;
  border-radius: var(--radius-md);
  overflow-x: auto;
  border: 1px solid var(--bg-surface-hover);
}

.tool-input code {
  font-family: var(--font-mono);
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text-secondary);
  line-height: 1.5;
}

.permission-options {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.option-btn {
  padding: 8px 16px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  font-family: var(--font-sans);
  transition: all var(--duration-fast) var(--ease-snappy);
  display: flex;
  align-items: center;
  justify-content: center;
}

.option-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none !important;
}

/* Allow Button - Emerald */
.option-btn.allow {
  background: #10b981;
  color: white;
  box-shadow: 0 1px 2px rgba(16, 185, 129, 0.2);
}

.option-btn.allow:hover:not(:disabled) {
  background: #059669;
  transform: translateY(-1px);
  box-shadow: 0 4px 6px rgba(16, 185, 129, 0.3);
}

.option-btn.allow:active:not(:disabled) {
  transform: translateY(0);
}

/* Reject Button - Uses accent-error */
.option-btn.reject {
  background: transparent;
  border: 1px solid var(--bg-element);
  color: var(--text-secondary);
}

.option-btn.reject:hover:not(:disabled) {
  border-color: var(--accent-error);
  color: var(--accent-error);
  background: var(--bg-surface-hover);
}
</style>
