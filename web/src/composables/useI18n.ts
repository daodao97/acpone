import { ref, computed } from 'vue'

type Lang = 'en' | 'zh'

const storedLang = localStorage.getItem('acp-lang') as Lang | null
const currentLang = ref<Lang>(storedLang || 'en')

// Simple dictionary
const messages: Record<Lang, Record<string, string>> = {
    en: {
        // Settings
        'settings.title': 'Settings',
        'settings.agents': 'Agents Configuration',
        'settings.agents.desc': 'Available ACP agents and their settings.',
        'settings.default': 'Default',
        'settings.permission': 'Permission',
        'settings.permission.default': 'User Confirmation',
        'settings.permission.default.desc': 'Requires user approval for tool calls (recommended)',
        'settings.permission.bypass': 'Auto Approve',
        'settings.permission.bypass.desc': 'Automatically approves all tool calls (use with caution)',
        'settings.permission.mode': 'Permission Modes',
        'settings.permission.mode.desc': 'Explanation of different permission levels.',
        'settings.save': 'Save',
        'settings.saving': 'Saving...',
        'settings.env': 'Environment Variables',
        'settings.env.desc': 'Configure environment variables for this agent.',
        'settings.appearance': 'Appearance',
        'settings.theme': 'Interface Theme',
        'settings.theme.dark': 'Dark Mode ☾',
        'settings.theme.light': 'Light Mode ☀',
        'settings.language': 'Language',

        // Welcome
        'welcome.start': 'Start chatting!',
        'welcome.mention': 'Use @ to mention an agent',
        'welcome.select_workspace': 'Please select a workspace first',

        // Input
        'input.placeholder': 'Message... (Type @ to mention, / for commands)',

        // Sidebar
        'sidebar.new_chat': 'New Chat',
        'sidebar.settings': 'Settings',
    },
    zh: {
        // Settings
        'settings.title': '设置',
        'settings.agents': '智能体配置',
        'settings.agents.desc': '可用的 ACP 智能体及其设置。',
        'settings.default': '默认',
        'settings.permission': '权限模式',
        'settings.permission.default': '用户确认',
        'settings.permission.default.desc': '工具调用需要用户批准（推荐）',
        'settings.permission.bypass': '自动批准',
        'settings.permission.bypass.desc': '自动批准所有工具调用（请谨慎使用）',
        'settings.permission.mode': '权限模式说明',
        'settings.permission.mode.desc': '不同权限级别的详细说明。',
        'settings.save': '保存',
        'settings.saving': '保存中...',
        'settings.env': '环境变量',
        'settings.env.desc': '配置该智能体的环境变量。',
        'settings.appearance': '外观设置',
        'settings.theme': '界面主题',
        'settings.theme.dark': '深色模式 ☾',
        'settings.theme.light': '浅色模式 ☀',
        'settings.language': '语言设置',

        // Welcome
        'welcome.start': '开始对话！',
        'welcome.mention': '使用 @ 呼叫智能体',
        'welcome.select_workspace': '请先选择一个工作区',

        // Input
        'input.placeholder': '输入消息... (输入 @ 呼叫智能体, / 使用命令)',

        // Sidebar
        'sidebar.new_chat': '新对话',
        'sidebar.settings': '设置',
    }
}

export function useI18n() {
    const t = (key: string) => {
        return messages[currentLang.value][key] || key
    }

    const setLang = (lang: Lang) => {
        currentLang.value = lang
        localStorage.setItem('acp-lang', lang)
    }

    const toggleLang = () => {
        const newLang = currentLang.value === 'en' ? 'zh' : 'en'
        setLang(newLang)
    }

    return {
        currentLang: computed(() => currentLang.value),
        t,
        toggleLang,
        setLang
    }
}
