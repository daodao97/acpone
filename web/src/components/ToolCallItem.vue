<template>
  <div class="tool-call" :class="tool.status">
    <div class="tool-header" @click="toggleExpand">
      <span class="tool-name">{{ tool.toolName }}</span>
      <span v-if="displayTitle" class="tool-title">{{ displayTitle }}</span>
      <span class="expand-icon">{{ expanded ? '▼' : '▶' }}</span>
    </div>
    <div v-if="tool.description" class="tool-description">{{ tool.description }}</div>

    <!-- Expanded details -->
    <div v-if="expanded" class="tool-details">
      <div v-if="tool.input" class="detail-row">
        <span class="detail-label">Input:</span>
        <pre class="detail-value">{{ tool.input }}</pre>
      </div>
      <div v-if="tool.rawInput && tool.rawInput !== '{}'" class="detail-row">
        <span class="detail-label">Raw:</span>
        <pre class="detail-value">{{ formatJSON(tool.rawInput) }}</pre>
      </div>
    </div>

    <!-- Error message -->
    <pre v-if="tool.error && tool.status === 'error'" class="tool-error">{{ tool.error }}</pre>
    <!-- Output (result) -->
    <pre v-if="tool.output && tool.status === 'completed'" class="tool-output">{{ tool.output }}</pre>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { ToolCall } from '../types'

const props = defineProps<{
  tool: ToolCall
}>()

const expanded = ref(false)

function toggleExpand() {
  expanded.value = !expanded.value
}

// Display title: prefer title field, fallback to input
const displayTitle = computed(() => {
  // If title exists and is not just the toolCallId, use it
  if (props.tool.title && !props.tool.title.startsWith('toolu_')) {
    return props.tool.title
  }
  // Fallback to input if available
  if (props.tool.input) {
    return props.tool.input.length > 80 ? props.tool.input.slice(0, 80) + '...' : props.tool.input
  }
  return ''
})

function formatJSON(str: string): string {
  try {
    const obj = JSON.parse(str)
    return JSON.stringify(obj, null, 2)
  } catch {
    return str
  }
}
</script>

<style scoped>
.tool-call {
  position: relative;
  padding-left: 14px;
  margin: 2px 0;
  font-family: var(--font-mono);
  font-size: 13px;
}

/* Vertical Tree Line */
.tool-call::before {
  content: '';
  position: absolute;
  left: 5px;
  top: 18px;
  bottom: 0;
  width: 1px;
  background-color: var(--bg-element);
}

/* Hide line if no output/error/details */
.tool-call:not(:has(.tool-output)):not(:has(.tool-error)):not(:has(.tool-details))::before {
  display: none;
}

/* Status Dot */
.tool-header::before {
  content: '';
  position: absolute;
  left: 0;
  top: 5px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: var(--text-tertiary);
  z-index: 1;
}

.tool-call.completed .tool-header::before {
  background-color: #5cb85c;
}

.tool-call.error .tool-header::before {
  background-color: #d9534f;
}

.tool-call.pending .tool-header::before {
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.tool-header {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding-bottom: 2px;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tool-header:hover {
  opacity: 0.8;
}

.tool-name {
  font-weight: 700;
  color: var(--text-primary);
  flex-shrink: 0;
}

.tool-title {
  color: var(--text-secondary);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.expand-icon {
  font-size: 8px;
  color: var(--text-tertiary);
  margin-left: auto;
  flex-shrink: 0;
}

.tool-description {
  color: var(--text-tertiary);
  font-size: 11px;
  margin-top: 2px;
  font-style: italic;
}

.tool-details {
  margin-top: 4px;
  padding: 8px;
  background: var(--bg-surface);
  border-radius: var(--radius-sm);
  border: 1px solid var(--bg-element);
}

.detail-row {
  margin-bottom: 6px;
}

.detail-row:last-child {
  margin-bottom: 0;
}

.detail-label {
  font-size: 10px;
  color: var(--text-tertiary);
  text-transform: uppercase;
  display: block;
  margin-bottom: 2px;
}

.detail-value {
  font-size: 11px;
  color: var(--text-secondary);
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

/* Tree Connector (L-shape) for content */
.tool-output,
.tool-error {
  position: relative;
  margin-left: 8px;
  padding-left: 20px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-tertiary);
  white-space: pre-wrap;
  word-break: break-all;
  margin-top: 2px;
}

/* The horizontal line of the L */
.tool-output::before,
.tool-error::before {
  content: '';
  position: absolute;
  left: -9px;
  top: 0.7em;
  width: 16px;
  height: 1px;
  background-color: var(--bg-element);
}

.tool-error {
  color: #ff8888;
}

.tool-output {
  max-height: 300px;
  overflow-y: auto;
  scrollbar-width: none;
}
</style>
