<template>
  <AdminPageLayout>
    <template #page-header>
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">授权分享管理</h1>
        <p class="text-gray-600 dark:text-gray-400">登记授权证据，并管理由本站账号创建的合规分享链接与转存任务。</p>
      </div>
    </template>

    <template #content>
      <div class="space-y-6">
        <n-card title="选择资源" size="small">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
            <n-input-number v-model:value="resourceId" :min="1" :show-button="false" placeholder="资源 ID" class="sm:w-48" />
            <n-button type="primary" :loading="loading" @click="loadResource">加载</n-button>
            <span v-if="resource" class="text-sm text-gray-600 dark:text-gray-300">#{{ resource.id }} · {{ resource.title }}</span>
          </div>
          <p class="mt-3 text-xs text-gray-500">仅在授权有效、同平台且账号有效时才能创建任务；本页不会显示或分发原始资源链接。</p>
        </n-card>

        <template v-if="resource">
          <div class="grid gap-6 xl:grid-cols-2">
            <n-card title="资源授权" size="small">
              <n-form label-placement="top" :model="authorizationForm">
                <div class="grid gap-3 sm:grid-cols-2">
                  <n-form-item label="授权状态"><n-select v-model:value="authorizationForm.status" :options="authorizationStatusOptions" /></n-form-item>
                  <n-form-item label="证据类型"><n-input v-model:value="authorizationForm.evidence_type" placeholder="如：license / consent" /></n-form-item>
                </div>
                <n-form-item label="证据引用"><n-input v-model:value="authorizationForm.evidence_ref" placeholder="内部工单、存证编号或授权文件引用" /></n-form-item>
                <n-form-item label="保留至（可选）"><n-date-picker v-model:formatted-value="authorizationForm.retention_until" value-format="yyyy-MM-dd'T'HH:mm:ssZ" type="datetime" clearable class="w-full" /></n-form-item>
                <div class="flex items-center gap-3">
                  <n-button type="primary" :loading="savingAuthorization" @click="saveAuthorization">保存授权</n-button>
                  <n-tag v-if="authorization?.verified_by" size="small">核验人：{{ authorization.verified_by }}</n-tag>
                </div>
              </n-form>
            </n-card>

            <n-card title="目标账号健康与容量" size="small">
              <div class="space-y-3">
                <n-select v-model:value="transferForm.pan_id" :options="panOptions" placeholder="选择目标平台" @update:value="transferForm.ck_id = null" />
                <n-select v-model:value="transferForm.ck_id" :options="accountOptions" placeholder="选择同平台有效账号" />
                <div v-if="selectedAccount" class="rounded bg-gray-50 p-3 text-sm dark:bg-gray-800">
                  <div class="flex items-center justify-between"><span>{{ selectedAccount.username || `账号 #${selectedAccount.id}` }}</span><n-tag :type="selectedAccount.is_valid ? 'success' : 'error'">{{ selectedAccount.is_valid ? '有效' : '无效' }}</n-tag></div>
                  <div class="mt-2 text-gray-600 dark:text-gray-300">可用 {{ formatBytes(selectedAccount.left_space) }} / 总计 {{ formatBytes(selectedAccount.space) }}</div>
                </div>
                <div class="flex gap-3"><n-button :disabled="!selectedAccount" :loading="refreshingAccount" @click="refreshAccount">刷新容量</n-button><n-button type="primary" :disabled="!transferForm.pan_id || !transferForm.ck_id" :loading="creatingTask" @click="createTask">创建转存任务</n-button></div>
              </div>
            </n-card>
          </div>

          <n-card title="自有分享链接" size="small">
            <template #header-extra><n-button size="small" :loading="checkingShares" @click="checkShares">检测有效性</n-button></template>
            <n-data-table :columns="shareColumns" :data="shares" :loading="loadingShares" :pagination="false" :scroll-x="850" />
          </n-card>

          <n-card title="授权转存任务" size="small">
            <template #header-extra><n-button size="small" @click="loadTasks">刷新</n-button></template>
            <n-data-table :columns="taskColumns" :data="tasks" :loading="loadingTasks" :pagination="false" :scroll-x="800" />
          </n-card>
        </template>
      </div>
    </template>
  </AdminPageLayout>
</template>

<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { NButton, NTag, useMessage } from 'naive-ui'
import { useAuthorizedShareApi, useCksApi, usePanApi, useResourceApi, useTaskApi } from '~/composables/useApi'

definePageMeta({ layout: 'admin', middleware: ['auth'] })

const message = useMessage()
const authorizedShareApi = useAuthorizedShareApi()
const resourceApi = useResourceApi()
const panApi = usePanApi()
const cksApi = useCksApi()
const taskApi = useTaskApi()
const resourceId = ref<number | null>(null)
const resource = ref<any>(null)
const authorization = ref<any>(null)
const shares = ref<any[]>([])
const tasks = ref<any[]>([])
const pans = ref<any[]>([])
const accounts = ref<any[]>([])
const loading = ref(false)
const loadingShares = ref(false)
const loadingTasks = ref(false)
const savingAuthorization = ref(false)
const creatingTask = ref(false)
const checkingShares = ref(false)
const refreshingAccount = ref(false)
const authorizationForm = ref({ status: 'pending', evidence_type: '', evidence_ref: '', retention_until: null as string | null })
const transferForm = ref({ pan_id: null as number | null, ck_id: null as number | null, channel: 'admin' })

const authorizationStatusOptions = [
  { label: '待核验', value: 'pending' }, { label: '有效', value: 'active' }, { label: '已撤销', value: 'revoked' }, { label: '已到期', value: 'expired' },
]
const panOptions = computed(() => pans.value.map((pan: any) => ({ label: pan.remark || pan.name, value: pan.id })))
const accountOptions = computed(() => accounts.value.filter((account: any) => account.pan_id === transferForm.value.pan_id).map((account: any) => ({ label: `${account.username || `账号 #${account.id}`} · ${account.is_valid ? '有效' : '无效'} · ${formatBytes(account.left_space)}`, value: account.id, disabled: !account.is_valid })))
const selectedAccount = computed(() => accounts.value.find((account: any) => account.id === transferForm.value.ck_id))

const formatBytes = (value?: number) => {
  const bytes = Number(value || 0)
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / (1024 ** index)).toFixed(index < 2 ? 0 : 2)} ${units[index]}`
}
const formatTime = (value?: string) => value ? new Date(value).toLocaleString('zh-CN') : '—'

const shareColumns = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '目标', key: 'pan_id', render: (row: any) => `${pans.value.find((pan: any) => pan.id === row.pan_id)?.remark || `平台 #${row.pan_id}`} / 账号 #${row.ck_id}` },
  { title: '状态', key: 'status', render: (row: any) => h(NTag, { type: row.status === 'active' ? 'success' : 'error', size: 'small' }, { default: () => row.status }) },
  { title: '检测结果', key: 'last_check_status', render: (row: any) => row.last_check_status || '未检测' },
  { title: '最近检测', key: 'last_checked_at', render: (row: any) => formatTime(row.last_checked_at) },
  { title: '失效原因', key: 'last_check_fail_reason', ellipsis: { tooltip: true }, render: (row: any) => row.last_check_fail_reason || '—' },
]
const taskColumns = [
  { title: 'ID', key: 'id', width: 70 }, { title: '状态', key: 'status', render: (row: any) => h(NTag, { type: row.status === 'completed' ? 'success' : row.status === 'failed' ? 'error' : 'warning', size: 'small' }, { default: () => row.status }) },
  { title: '进度', key: 'processed_items', render: (row: any) => `${row.processed_items || 0}/${row.total_items || 0}` },
  { title: '信息', key: 'description', ellipsis: { tooltip: true } }, { title: '创建时间', key: 'created_at', render: (row: any) => formatTime(row.created_at) },
  { title: '操作', key: 'actions', width: 100, render: (row: any) => row.status === 'failed' ? h(NButton, { size: 'small', type: 'warning', onClick: () => retryTask(row.id) }, { default: () => '重试' }) : '—' },
]

async function loadSupportData() {
  const [panResult, accountResult] = await Promise.all([panApi.getPans(), cksApi.getCks()])
  pans.value = Array.isArray(panResult) ? panResult : (panResult as any)?.data || []
  accounts.value = Array.isArray(accountResult) ? accountResult : (accountResult as any)?.data || []
}
async function loadResource() {
  if (!resourceId.value) return message.warning('请输入资源 ID')
  loading.value = true
  try {
    const [resourceResult, authResult] = await Promise.all([resourceApi.getResource(resourceId.value), authorizedShareApi.getAuthorization(resourceId.value)])
    resource.value = resourceResult
    authorization.value = (authResult as any)?.authorization || null
    authorizationForm.value = authorization.value ? { status: authorization.value.status, evidence_type: authorization.value.evidence_type, evidence_ref: authorization.value.evidence_ref, retention_until: authorization.value.retention_until } : { status: 'pending', evidence_type: '', evidence_ref: '', retention_until: null }
    await Promise.all([loadSupportData(), loadShares(), loadTasks()])
  } catch (error: any) { resource.value = null; message.error(error?.message || '加载资源失败') } finally { loading.value = false }
}
async function loadShares() { if (!resourceId.value) return; loadingShares.value = true; try { const result: any = await authorizedShareApi.getOwnedShares(resourceId.value); shares.value = result?.list || (Array.isArray(result) ? result : []) } catch (error: any) { message.error(error?.message || '加载分享链接失败') } finally { loadingShares.value = false } }
async function loadTasks() { if (!resourceId.value) return; loadingTasks.value = true; try { const result: any = await taskApi.getTasks({ page: 1, pageSize: 100, taskType: 'authorized_transfer' }); tasks.value = (result?.tasks || []).filter((task: any) => { try { const config = JSON.parse(task.config || '{}'); return config.resource_id === resourceId.value || task.title?.includes(`resource ${resourceId.value}`) } catch { return task.title?.includes(`resource ${resourceId.value}`) } }) } catch (error: any) { message.error(error?.message || '加载任务失败') } finally { loadingTasks.value = false } }
async function saveAuthorization() { if (!resourceId.value || !authorizationForm.value.evidence_type || !authorizationForm.value.evidence_ref) return message.warning('请填写证据类型和引用') ; savingAuthorization.value = true; try { await authorizedShareApi.saveAuthorization(resourceId.value, authorizationForm.value); message.success('授权已保存'); const result: any = await authorizedShareApi.getAuthorization(resourceId.value); authorization.value = result.authorization } catch (error: any) { message.error(error?.message || '保存授权失败') } finally { savingAuthorization.value = false } }
async function refreshAccount() { if (!selectedAccount.value) return; refreshingAccount.value = true; try { await cksApi.refreshCapacity(selectedAccount.value.id); await loadSupportData(); message.success('账号容量已刷新') } catch (error: any) { message.error(error?.message || '刷新容量失败') } finally { refreshingAccount.value = false } }
async function createTask() { if (!resourceId.value || !transferForm.value.pan_id || !transferForm.value.ck_id) return; creatingTask.value = true; try { const result: any = await authorizedShareApi.createTransferTask(resourceId.value, transferForm.value); message.success(result?.reused ? '已复用有效自有分享链接' : '转存任务已创建'); await Promise.all([loadShares(), loadTasks()]) } catch (error: any) { message.error(error?.message || '创建任务失败') } finally { creatingTask.value = false } }
async function checkShares() { if (!resourceId.value) return; checkingShares.value = true; try { await authorizedShareApi.checkOwnedShares(resourceId.value); message.success('检测已完成'); await loadShares() } catch (error: any) { message.error(error?.message || '检测失败') } finally { checkingShares.value = false } }
async function retryTask(taskId: number) { if (!resourceId.value) return; try { await authorizedShareApi.retryTransferTask(resourceId.value, taskId); message.success('任务已重新排队'); await loadTasks() } catch (error: any) { message.error(error?.message || '重试失败') } }
</script>
