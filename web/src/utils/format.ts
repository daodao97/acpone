export function formatTime(timestamp: number): string {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return 'now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h`
  return date.toLocaleDateString()
}

export function getToolIcon(toolName: string): string {
  const icons: Record<string, string> = {
    Bash: '💻',
    Read: '📄',
    Write: '✏️',
    Edit: '✏️',
    Glob: '🔍',
    Grep: '🔍',
    Terminal: '💻',
  }
  return icons[toolName] || '🔧'
}
