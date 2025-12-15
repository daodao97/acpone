import { ref } from 'vue'

type Theme = 'light' | 'dark'

const currentTheme = ref<Theme>('dark')

function initTheme() {
    const saved = localStorage.getItem('acp-theme') as Theme | null
    if (saved) {
        currentTheme.value = saved
    } else {
        // Check system preference
        if (window.matchMedia('(prefers-color-scheme: light)').matches) {
            currentTheme.value = 'light'
        } else {
            currentTheme.value = 'dark'
        }
    }
    applyTheme(currentTheme.value)
}

function applyTheme(theme: Theme) {
    document.documentElement.setAttribute('data-theme', theme)
}

function toggleTheme() {
    const newTheme = currentTheme.value === 'dark' ? 'light' : 'dark'
    currentTheme.value = newTheme
    applyTheme(newTheme)
    localStorage.setItem('acp-theme', newTheme)
}

function setTheme(theme: Theme) {
    currentTheme.value = theme
    applyTheme(theme)
    localStorage.setItem('acp-theme', theme)
}

// Initialize immediately
initTheme()

export function useTheme() {
    return {
        currentTheme,
        toggleTheme,
        setTheme
    }
}
