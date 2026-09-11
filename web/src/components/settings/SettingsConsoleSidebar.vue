<script setup lang="ts">
import { computed } from 'vue'

export type SettingsCategory =
  | 'sources'
  | 'providers'
  | 'network'
  | 'renaming'
  | 'automation'
  | 'integrations'
  | 'interface'
  | 'system'

const props = defineProps<{
  activeCategory: SettingsCategory
  sourceCount: number
  pendingApprovalsCount?: number
  locale: string
  labels: Record<string, string>
}>()

const emit = defineEmits<{
  selectCategory: [category: SettingsCategory]
}>()

const zh = computed(() => props.locale === 'zh-CN')

interface MenuItem {
  id: SettingsCategory
  titleZh: string
  titleEn: string
  descZh: string
  descEn: string
  icon: string
  badge?: number | string
}

const menuItems = computed<MenuItem[]>(() => [
  {
    id: 'sources',
    titleZh: '媒体源与挂载',
    titleEn: 'Library Sources',
    descZh: '目录挂载与扫描策略',
    descEn: 'Storage & scan schedule',
    icon: 'storage',
    badge: props.sourceCount > 0 ? props.sourceCount : undefined,
  },
  {
    id: 'providers',
    titleZh: '元数据刮削源',
    titleEn: 'Metadata Providers',
    descZh: 'TMDb 与 Fanart.tv 凭证',
    descEn: 'API keys & languages',
    icon: 'api',
  },
  {
    id: 'network',
    titleZh: '网络与代理',
    titleEn: 'Network & Proxy',
    descZh: '出站代理与 NO_PROXY',
    descEn: 'HTTP / SOCKS5 proxy',
    icon: 'router',
  },
  {
    id: 'renaming',
    titleZh: '重命名规则',
    titleEn: 'Rename Patterns',
    descZh: '电影与剧集默认格式',
    descEn: 'Movie & TV defaults',
    icon: 'drive_file_rename_outline',
  },
  {
    id: 'automation',
    titleZh: '自动化安全审批',
    titleEn: 'File Approvals',
    descZh: '文件写入与计划审批',
    descEn: 'Write plan review',
    icon: 'verified_user',
    badge: (props.pendingApprovalsCount ?? 0) > 0 ? props.pendingApprovalsCount : undefined,
  },
  {
    id: 'integrations',
    titleZh: '外部集成与 API',
    titleEn: 'Integrations & MCP',
    descZh: 'Webhook 与 MCP 令牌',
    descEn: 'Webhooks & MCP tokens',
    icon: 'hub',
  },
  {
    id: 'interface',
    titleZh: '界面与偏好',
    titleEn: 'Interface & Theme',
    descZh: '语言与深浅主题',
    descEn: 'Language & theme',
    icon: 'palette',
  },
  {
    id: 'system',
    titleZh: '系统与运行环境',
    titleEn: 'System & Docker',
    descZh: '版本、数据库与健康诊断',
    descEn: 'Runtime & database',
    icon: 'dns',
  },
])
</script>

<template>
  <aside class="settings-console-sidebar" :aria-label="zh ? '设置导航' : 'Settings navigation'">
    <!-- Console Brand Header -->
    <div class="sidebar-header">
      <div class="header-badge">
        <span class="dot dot-ok"></span>
        <span class="eyebrow">CONSOLE</span>
      </div>
      <h2 class="sidebar-title">{{ zh ? '设置控制台' : 'Settings Console' }}</h2>
      <p class="sidebar-subtitle">{{ zh ? '系统配置与集成中心' : 'Configuration & Management' }}</p>
    </div>

    <!-- Category Menu List -->
    <nav class="menu-list">
      <button
        v-for="item in menuItems"
        :key="item.id"
        type="button"
        class="menu-item"
        :class="{ 'menu-item--active': activeCategory === item.id }"
        @click="emit('selectCategory', item.id)"
      >
        <div class="item-icon-box">
          <svg v-if="item.icon === 'storage'" class="item-svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M2 20h20v-4H2v4zm2-3h2v2H4v-2zM2 4v4h20V4H2zm4 3H4V5h2v2zm-4 7h20v-4H2v4zm2-3h2v2H4v-2z"/>
          </svg>
          <svg v-else-if="item.icon === 'api'" class="item-svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M14 12l-2 2-2-2 2-2 2 2zm-2-6l4 4-4 4-4-4 4-4zm8 8l-2-2-2 2 2 2 2-2zm-6 2l-2 2-2-2 2-2 2 2zm-6-2l-2-2-2 2 2 2 2-2zm-4-4l4-4-4-4-4 4 4 4zm16 0l4-4-4-4-4 4 4 4z"/>
          </svg>
          <svg v-else-if="item.icon === 'router'" class="item-svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20.2 5.9l.8-.8C19.6 3.7 17.8 3 16 3s-3.6.7-5 2.1l.8.8C13 4.8 14.5 4.2 16 4.2s3 .6 4.2 1.7zm2.8-2.8l.8-.8C21.6.8 18.9 0 16 0S10.4.8 8.2 2.3l.8.8C10.8 1.9 13.3 1.2 16 1.2s5.2.7 7 1.9zM19 13h-2V9h-2v4H5c-1.1 0-2 .9-2 2v4c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2v-4c0-1.1-.9-2-2-2zM6 18H4v-2h2v2zm3 0H7v-2h2v2zm3 0h-2v-2h2v2zm7 0h-2v-2h2v2z"/>
          </svg>
          <svg v-else-if="item.icon === 'palette'" class="item-svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 3c-4.97 0-9 4.03-9 9 0 2.12.74 4.07 1.97 5.61L4.35 19c-.39.39-.39 1.02 0 1.41.39.39 1.02.39 1.41 0l1.9-1.9C9.07 19.38 10.48 20 12 20c4.97 0 9-4.03 9-9s-4.03-9-9-9zm-5 9c-.83 0-1.5-.67-1.5-1.5S6.17 9 7 9s1.5.67 1.5 1.5S7.83 12 7 12zm3-4c-.83 0-1.5-.67-1.5-1.5S9.17 5 10 5s1.5.67 1.5 1.5S10.83 8 10 8zm4 0c-.83 0-1.5-.67-1.5-1.5S13.17 5 14 5s1.5.67 1.5 1.5S14.83 8 14 8zm3 4c-.83 0-1.5-.67-1.5-1.5S16.17 9 17 9s1.5.67 1.5 1.5S17.83 12 17 12z"/>
          </svg>
          <svg v-else-if="item.icon === 'hub'" class="item-svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
          </svg>
          <svg v-else-if="item.icon === 'verified_user'" class="item-svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm-2 16l-4-4 1.41-1.41L10 14.17l6.59-6.59L18 9l-8 8z"/>
          </svg>
          <svg v-else class="item-svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"/>
          </svg>
        </div>

        <div class="item-text">
          <span class="item-title">{{ zh ? item.titleZh : item.titleEn }}</span>
          <span class="item-desc">{{ zh ? item.descZh : item.descEn }}</span>
        </div>

        <span v-if="item.badge !== undefined" class="item-badge spec-pill">
          {{ item.badge }}
        </span>
      </button>
    </nav>
  </aside>
</template>

<style scoped>
.settings-console-sidebar {
  width: 260px;
  min-width: 260px;
  height: 100%;
  background: var(--surface-container-low, #151b2d);
  border-right: 1px solid var(--outline-variant, #2e3447);
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  flex-shrink: 0;
  user-select: none;
}

.sidebar-header {
  padding: 20px 18px 16px;
  border-bottom: 1px solid var(--outline-variant, #2e3447);
  background: var(--surface-container, #191f31);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.header-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.eyebrow {
  margin: 0;
  font-family: var(--font-data, monospace);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--primary, #c0c1ff);
}

.sidebar-title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--on-surface, #dce1fb);
  letter-spacing: -0.01em;
}

.sidebar-subtitle {
  margin: 0;
  font-size: 11px;
  color: var(--on-surface-variant, #c7c4d7);
}

.menu-list {
  flex: 1;
  overflow-y: auto;
  padding: 10px 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid transparent;
  border-radius: var(--radius-md, 0.375rem);
  background: transparent;
  color: var(--on-surface-variant, #c7c4d7);
  text-align: left;
  cursor: pointer;
  transition: all 0.15s ease;
  position: relative;
}

.menu-item:hover {
  background: var(--surface-container, #191f31);
  color: var(--on-surface, #dce1fb);
}

.menu-item--active {
  background: var(--surface-container-highest, #2e3447);
  color: var(--primary, #c0c1ff);
  border-color: rgba(192, 193, 255, 0.2);
  font-weight: 600;
}

.menu-item--active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 6px;
  bottom: 6px;
  width: 3px;
  background: var(--primary, #c0c1ff);
  border-radius: 0 2px 2px 0;
}

.item-icon-box {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.item-svg {
  width: 18px;
  height: 18px;
  color: currentColor;
}

.item-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.item-title {
  font-size: 13px;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-desc {
  font-size: 10px;
  color: var(--outline, #908fa0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.menu-item--active .item-desc {
  color: var(--on-surface-variant, #c7c4d7);
}

.item-badge {
  font-size: 10px;
  padding: 1px 6px;
  background: var(--surface-container-high, #23293c);
  color: var(--primary, #c0c1ff);
  border-radius: 9999px;
}

@media (max-width: 860px) {
  .settings-console-sidebar {
    width: 100%;
    height: auto;
    min-width: 0;
    border-right: none;
    border-bottom: 1px solid var(--outline-variant, #2e3447);
  }

  .menu-list {
    flex-direction: row;
    overflow-x: auto;
    padding: 8px;
  }

  .menu-item {
    flex-shrink: 0;
    width: auto;
  }

  .item-desc {
    display: none;
  }
}
</style>
