<template>
  <div class="main-shell">
    <div class="global-actions">
      <p class="global-eyebrow">{{ t('components.mcp.hero.eyebrow') }}</p>
      <button class="ghost-icon" :data-tooltip="t('components.mcp.actions.back')" @click="goHome">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path
            d="M15 18l-6-6 6-6"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
      <button
        class="ghost-icon"
        :data-tooltip="t('components.mcp.actions.refresh')"
        :disabled="loading"
        @click="loadAll"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true" :class="{ spin: loading }">
          <path
            d="M20.5 8a8.5 8.5 0 10-2.38 7.41"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
          <path
            d="M20.5 4v4h-4"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
    </div>

    <div class="contrib-page mcp-page">
      <section class="contrib-hero">
        <h1>{{ t('components.mcp.hero.title') }}</h1>
        <p class="lead">{{ t('components.mcp.hero.lead') }}</p>
      </section>

      <!-- CLI 接入区 -->
      <section class="automation-section">
        <h2 class="mac-section-title">{{ t('components.mcp.cli.title') }}</h2>
        <div class="mac-panel">
          <div class="mcp-endpoint-row">
            <span class="mcp-endpoint-label">{{ t('components.mcp.cli.endpoint') }}</span>
            <code class="mcp-endpoint-value">{{ endpoint }}</code>
            <BaseButton variant="outline" size="sm" @click="copyEndpoint">
              {{ t('components.mcp.cli.copy') }}
            </BaseButton>
          </div>
          <div v-for="platform in platforms" :key="platform.id" class="mcp-cli-row">
            <span class="mcp-cli-name">{{ platform.label }}</span>
            <span
              class="mcp-inject-badge"
              :class="{ injected: injections[platform.id] === true }"
            >
              {{
                injections[platform.id] === null
                  ? '...'
                  : injections[platform.id]
                    ? t('components.mcp.cli.injected')
                    : t('components.mcp.cli.notInjected')
              }}
            </span>
            <BaseButton
              variant="outline"
              size="sm"
              :disabled="injecting[platform.id]"
              @click="toggleInjection(platform.id)"
            >
              {{
                injections[platform.id]
                  ? t('components.mcp.cli.remove')
                  : t('components.mcp.cli.inject')
              }}
            </BaseButton>
          </div>
          <details class="mcp-help">
            <summary>{{ t('components.mcp.cli.helpTitle') }}</summary>
            <ul>
              <li>{{ t('components.mcp.cli.helpRestart') }}</li>
              <li>{{ t('components.mcp.cli.helpRefresh') }}</li>
              <li>{{ t('components.mcp.cli.helpContext') }}</li>
              <li>{{ t('components.mcp.cli.helpCodexBackend') }}</li>
            </ul>
            <p class="mcp-help-manual">{{ t('components.mcp.cli.manualTitle') }}</p>
            <pre>claude mcp add --transport http code-switch {{ endpoint }} --header "Authorization: Bearer code-switch"</pre>
            <pre>[mcp_servers.code-switch]
url = "{{ endpoint }}"

[mcp_servers.code-switch.http_headers]
Authorization = "Bearer code-switch"</pre>
          </details>
        </div>
      </section>

      <!-- 服务列表区 -->
      <section class="automation-section">
        <div class="section-header">
          <h2 class="mac-section-title">
            {{ t('components.mcp.servers.title') }}
            <span v-if="totalTools > 0" class="mcp-total-tools">
              {{ t('components.mcp.servers.totalTools', { count: totalTools }) }}
            </span>
          </h2>
          <div class="section-controls">
            <BaseButton variant="outline" size="sm" @click="openCreate">
              {{ t('components.mcp.servers.add') }}
            </BaseButton>
          </div>
        </div>
        <div class="mac-panel">
          <p v-if="servers.length === 0" class="mcp-empty">{{ t('components.mcp.servers.empty') }}</p>
          <div v-for="server in servers" :key="server.id" class="mcp-server-row">
            <span class="mcp-status-dot" :class="statusClass(server.id)" :data-tooltip="statusTip(server.id)"></span>
            <div class="mcp-server-main">
              <div class="mcp-server-title">
                <span class="mcp-server-name">{{ server.name }}</span>
                <span class="mcp-type-badge">{{ server.type }}</span>
                <span v-if="statusOf(server.id)?.toolCount" class="mcp-tool-count">
                  {{ t('components.mcp.servers.toolCount', { count: statusOf(server.id)?.toolCount }) }}
                </span>
              </div>
              <div class="mcp-server-target">{{ targetOf(server) }}</div>
            </div>
            <label class="mac-switch sm" :data-tooltip="t('components.mcp.servers.toggle')">
              <input
                type="checkbox"
                :checked="server.enabled"
                @change="toggleEnabled(server)"
              />
              <span></span>
            </label>
            <button class="ghost-icon" :data-tooltip="t('components.mcp.servers.edit')" @click="openEdit(server)">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M16.5 4.5l3 3L8 19l-4 1 1-4L16.5 4.5z"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
            <button class="ghost-icon danger" :data-tooltip="t('components.mcp.servers.delete')" @click="askDelete(server)">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
          </div>
        </div>
      </section>
    </div>

    <McpEditorModal
      :open="editorState.open"
      :editing="editorState.form.id !== 0"
      :form="editorState.form"
      :error="editorState.error"
      :saving="editorState.saving"
      :testing="editorState.testing"
      :test-result="editorState.testResult"
      @close="closeEditor"
      @submit="submitEditor"
      @test="runTest"
    />

    <BaseModal
      :open="confirmState.open"
      variant="confirm"
      :title="t('components.mcp.confirm.title')"
      @close="confirmState.open = false"
    >
      <p>{{ t('components.mcp.confirm.body', { name: confirmState.target?.name ?? '' }) }}</p>
      <div class="confirm-actions">
        <BaseButton variant="outline" size="sm" :disabled="confirmState.busy" @click="confirmState.open = false">
          {{ t('components.mcp.form.cancel') }}
        </BaseButton>
        <BaseButton variant="danger" size="sm" :disabled="confirmState.busy" @click="confirmDelete">
          {{ t('components.mcp.confirm.delete') }}
        </BaseButton>
      </div>
    </BaseModal>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import BaseButton from '../common/BaseButton.vue'
import BaseModal from '../common/BaseModal.vue'
import McpEditorModal from './McpEditorModal.vue'
import { MCP_PLATFORMS, useMcpServers } from './useMcpServers'
import { showToast } from '../../utils/toast'
import type { McpServerRecord } from '../../services/mcp'

const { t } = useI18n()
const router = useRouter()

const {
  servers,
  statuses,
  endpoint,
  loading,
  injections,
  injecting,
  editorState,
  confirmState,
  totalTools,
  loadAll,
  refreshStatuses,
  openCreate,
  openEdit,
  closeEditor,
  submitEditor,
  runTest,
  toggleEnabled,
  askDelete,
  confirmDelete,
  toggleInjection,
} = useMcpServers(t)

const platforms = MCP_PLATFORMS

const goHome = () => {
  void router.push('/')
}

const copyEndpoint = async () => {
  try {
    await navigator.clipboard.writeText(endpoint.value)
    showToast(t('components.mcp.messages.copied'), 'success')
  } catch {
    showToast(t('components.mcp.messages.copyFailed'), 'error')
  }
}

const statusOf = (id: number) => statuses.value[id]

const statusClass = (id: number) => {
  const st = statuses.value[id]
  if (!st) return 'unknown'
  return st.state // connected / connecting / failed / disabled
}

const statusTip = (id: number) => {
  const st = statuses.value[id]
  if (!st) return ''
  const key = `components.mcp.status.${st.state}`
  const label = t(key)
  return st.error ? `${label}: ${st.error}` : label
}

const targetOf = (server: McpServerRecord) =>
  server.type === 'stdio'
    ? [server.command, ...(server.args ?? [])].join(' ')
    : server.url

let pollTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  void loadAll()
  // 状态轮询：聚合器重连退避语义下 5s 粒度足够（全仓前端无 EventsOn 使用，与 Sessions 页一致）
  pollTimer = setInterval(() => void refreshStatuses(), 5000)
})
onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.mcp-page {
  gap: 18px;
}

.mcp-endpoint-row,
.mcp-cli-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 4px;
}

.mcp-cli-row + .mcp-cli-row,
.mcp-endpoint-row + .mcp-cli-row {
  border-top: 1px solid var(--mac-border, rgba(0, 0, 0, 0.06));
}

.mcp-endpoint-label {
  color: var(--mac-text-secondary);
  font-size: 0.85rem;
}

.mcp-endpoint-value {
  flex: 1;
  font-size: 0.85rem;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mcp-cli-name {
  width: 110px;
  font-weight: 500;
}

.mcp-inject-badge {
  flex: 1;
  font-size: 0.8rem;
  color: var(--mac-text-secondary);
}

.mcp-inject-badge.injected {
  color: var(--mac-success, #1a7f37);
}

.mcp-help {
  padding: 10px 4px;
  border-top: 1px solid var(--mac-border, rgba(0, 0, 0, 0.06));
  font-size: 0.82rem;
  color: var(--mac-text-secondary);
}

.mcp-help pre {
  background: var(--mac-panel-alt, rgba(0, 0, 0, 0.04));
  padding: 8px;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 0.78rem;
}

.mcp-help-manual {
  margin: 8px 0 4px;
  font-weight: 600;
}

.mcp-total-tools {
  font-size: 0.78rem;
  font-weight: 400;
  color: var(--mac-text-secondary);
  margin-left: 8px;
}

.mcp-empty {
  padding: 18px 4px;
  color: var(--mac-text-secondary);
  text-align: center;
}

.mcp-server-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 4px;
}

.mcp-server-row + .mcp-server-row {
  border-top: 1px solid var(--mac-border, rgba(0, 0, 0, 0.06));
}

.mcp-status-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--mac-text-secondary);
  flex-shrink: 0;
}

.mcp-status-dot.connected {
  background: var(--mac-success, #1a7f37);
}

.mcp-status-dot.connecting {
  background: #d29922;
  animation: pulse 1.2s ease-in-out infinite;
}

.mcp-status-dot.failed {
  background: var(--mac-danger, #d0342c);
}

.mcp-status-dot.disabled {
  background: var(--mac-text-secondary);
  opacity: 0.5;
}

.mcp-server-main {
  flex: 1;
  min-width: 0;
}

.mcp-server-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mcp-server-name {
  font-weight: 500;
}

.mcp-type-badge {
  font-size: 0.72rem;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--mac-panel-alt, rgba(0, 0, 0, 0.05));
  color: var(--mac-text-secondary);
}

.mcp-tool-count {
  font-size: 0.78rem;
  color: var(--mac-text-secondary);
}

.mcp-server-target {
  font-size: 0.78rem;
  color: var(--mac-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ghost-icon.danger:hover {
  color: var(--mac-danger, #d0342c);
}

.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes pulse {
  50% {
    opacity: 0.4;
  }
}
</style>
