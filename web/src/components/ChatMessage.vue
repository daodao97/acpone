<script setup lang="ts">
import MarkdownRender from 'markstream-vue'
import type { Message } from '../types'
import ToolCallItem from './ToolCallItem.vue'

defineProps<{
  message: Message
  hideAgentTag?: boolean
}>()

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
</script>

<template>
  <!-- Tool call message -->
  <ToolCallItem v-if="message.toolCall" :tool="message.toolCall" />

  <!-- Error message -->
  <div v-else-if="message.isError" class="message error">
    <span class="error-icon">⚠</span>
    <div class="error-content">{{ message.content }}</div>
  </div>

  <!-- Text message -->
  <div v-else class="message" :class="message.role">
    <span v-if="message.role === 'assistant' && message.agent && !hideAgentTag" class="agent-tag"
      :class="message.agent">
      {{ message.agent }}
    </span>
    <!-- File attachments for user messages -->
    <div v-if="message.role === 'user' && message.files && message.files.length > 0" class="message-files">
      <div v-for="file in message.files" :key="file.path" class="message-file">
        <svg class="file-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
          <polyline points="14 2 14 8 20 8"></polyline>
        </svg>
        <span class="file-name">{{ file.name }}</span>
        <span class="file-size">{{ formatFileSize(file.size) }}</span>
      </div>
    </div>
    <div class="content">
      <MarkdownRender :content="message.content" />
    </div>
  </div>
</template>

<style scoped>
.message {
  padding: 0;
  max-width: 100%;
  word-wrap: break-word;
}

/* User Message: Minimalist Bubble */
.message.user {
  align-self: flex-end;
  color: var(--text-primary);
  background: var(--bg-element);
  padding: 8px 12px;
  border-radius: var(--radius-lg);
  border-bottom-right-radius: 2px;
  /* Slight accent */
  max-width: 80%;
  font-size: 14px;
  border: 1px solid var(--bg-surface-hover);
  margin: 16px 0 16px;
  /* Separation from stream */
}

/* Assistant Message: The Void (No Bubble) */
.message.assistant {
  align-self: stretch;
  background: transparent;
  padding: 0;
  margin-bottom: 4px;
  /* Very tight spacing */
}

/* Error Message */
.message.error {
  background: rgba(207, 51, 51, 0.1);
  /* accent-error with opacity */
  border: 1px solid var(--accent-error);
  color: #ff8888;
  align-self: stretch;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border-radius: var(--radius-md);
  margin: 8px 0;
}

.error-icon {
  font-size: 16px;
  color: var(--accent-error);
  flex-shrink: 0;
  margin-top: 2px;
}

.error-content {
  font-size: 13px;
  line-height: 1.6;
  font-family: var(--font-mono);
}

.agent-tag {
  display: inline-block;
  font-size: 10px;
  font-weight: 700;
  padding: 2px 0;
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--text-tertiary);
}

/* Markdown Override (Scoped to message) */
.content {
  font-size: 14px;
  line-height: 1.6;
}

.content :deep(p),
.content :deep(.paragraph-node) {
  margin-top: 0.5em !important;
  margin-bottom: 0.5em !important;
} 


.content :deep(p:last-child),
.content :deep(.paragraph-node:last-child) {
  margin-bottom: 0 !important;
}

/* File attachments in user messages */
.message-files {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.message-file {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: var(--bg-surface);
  border-radius: var(--radius-sm);
  font-size: 12px;
  border: 1px solid var(--accent-subtle);
}

.file-icon {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.file-name {
  color: var(--text-primary);
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  color: var(--text-tertiary);
  font-size: 11px;
  flex-shrink: 0;
}
</style>
