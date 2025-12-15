<script setup lang="ts">
import MarkdownRender from 'markstream-vue'
import type { Message } from '../types'
import ToolCallItem from './ToolCallItem.vue'

defineProps<{
  message: Message
  hideAgentTag?: boolean
}>()
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

</style>
