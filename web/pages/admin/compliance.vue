<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">合规审计与运营看板</h1>
        <p class="text-gray-600 dark:text-gray-400">查看授权、分享链接、任务和 Provider 部署审批状态。</p>
      </div>
      <n-button :loading="loading" @click="loadDashboard">
        <template #icon><i class="fas fa-refresh"></i></template>
        刷新
      </n-button>
    </div>

    <n-alert v-if="errorMessage" type="error" :title="errorMessage" />

    <div v-if="dashboard" class="space-y-6">
      <div class="grid grid-cols-2 gap-4 md:grid-cols-4 xl:grid-cols-6">
        <n-card v-for="card in metricCards" :key="card.label" size="small">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ card.label }}</p>
          <p class="mt-2 text-2xl font-semibold" :class="card.className">{{ card.value }}</p>
        </n-card>
      </div>

      <n-card title="Provider 合规闸门">
        <n-data-table :columns="providerColumns" :data="dashboard.providers" :bordered="false" :single-line="false" />
      </n-card>

      <n-card title="审计口径">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          报告生成于 {{ formatTime(dashboard.generated_at) }}；“即将到期”按未来 {{ dashboard.expiring_within_days }} 天统计。
          本页只展示聚合计数和审批引用，不展示原始资源链接、分享链接或账号凭据。
        </p>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { h, computed, onMounted, ref } from 'vue'
import { NButton, NAlert, NCard, NDataTable, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useComplianceApi } from '~/composables/useApi'

interface ProviderStatus {
  provider: string
  implementation_available: boolean
  transfer_contract_available: boolean
  approval_configured: boolean
  eligible_for_authorized_transfer: boolean
  approval_reference?: string
  max_share_retention_days?: number
  reason?: string
}

interface ComplianceDashboard {
  generated_at: string
  expiring_within_days: number
  providers: ProviderStatus[]
  metrics: Record<string, number>
}

const { getDashboard } = useComplianceApi()
const dashboard = ref<ComplianceDashboard | null>(null)
const loading = ref(false)
const errorMessage = ref('')

const metricCards = computed(() => {
  const m = dashboard.value?.metrics || {}
  return [
    { label: '有效授权', value: m.active_authorizations || 0, className: 'text-green-600' },
    { label: '授权即将到期', value: m.authorizations_expiring_soon || 0, className: 'text-amber-600' },
    { label: '有效自有分享', value: m.active_owned_shares || 0, className: 'text-blue-600' },
    { label: '失效自有分享', value: m.invalid_owned_shares || 0, className: 'text-red-600' },
    { label: '待处理转存', value: m.pending_authorized_transfers || 0, className: 'text-purple-600' },
    { label: '失败转存', value: m.failed_authorized_transfers || 0, className: 'text-red-600' }
  ]
})

const providerColumns: DataTableColumns<ProviderStatus> = [
  { title: 'Provider', key: 'provider' },
  {
    title: '实现', key: 'implementation_available', render: (row) => h(NTag, { type: row.implementation_available ? 'success' : 'warning' }, { default: () => row.implementation_available ? '已注册' : '未实现' })
  },
  {
    title: '部署审批', key: 'approval_configured', render: (row) => h(NTag, { type: row.approval_configured ? 'success' : 'error' }, { default: () => row.approval_configured ? '已配置' : '未配置' })
  },
  {
    title: '授权转存', key: 'eligible_for_authorized_transfer', render: (row) => h(NTag, { type: row.eligible_for_authorized_transfer ? 'success' : 'error' }, { default: () => row.eligible_for_authorized_transfer ? '允许' : '阻断' })
  },
  { title: '保留天数', key: 'max_share_retention_days', render: (row) => row.max_share_retention_days || '-' },
  { title: '原因/引用', key: 'reason', render: (row) => row.eligible_for_authorized_transfer ? (row.approval_reference || '-') : (row.reason || '-') }
]

const formatTime = (value: string) => value ? new Date(value).toLocaleString() : '-'

const loadDashboard = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    dashboard.value = await getDashboard() as ComplianceDashboard
  } catch (error: any) {
    errorMessage.value = error?.message || '加载合规看板失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadDashboard)
</script>
