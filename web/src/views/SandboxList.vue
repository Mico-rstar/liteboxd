<template>
  <div class="sandbox-list">
    <t-card title="Sandboxes" :bordered="false">
      <template #actions>
        <t-button theme="primary" @click="showCreateDialog = true">
          创建 Sandbox
        </t-button>
      </template>

      <t-table
        :data="sandboxes"
        :columns="columns"
        :loading="loading"
        row-key="id"
        hover
      >
        <template #id="{ row }">
          <t-link theme="primary" @click="goToDetail(row.id)">
            {{ row.id }}
          </t-link>
        </template>
        <template #status="{ row }">
          <t-tag :theme="getStatusTheme(row.status)">
            {{ row.status }}
          </t-tag>
        </template>
        <template #created_at="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
        <template #expires_at="{ row }">
          {{ formatTime(row.expires_at) }}
        </template>
        <template #operation="{ row }">
          <t-space>
            <t-link theme="primary" @click="goToDetail(row.id)">详情</t-link>
            <t-popconfirm content="确定要删除该 Sandbox 吗？" @confirm="deleteSandbox(row.id)">
              <t-link theme="danger">删除</t-link>
            </t-popconfirm>
          </t-space>
        </template>
      </t-table>
    </t-card>

    <t-dialog
      v-model:visible="showCreateDialog"
      header="创建 Sandbox"
      :confirm-btn="{ content: '创建', loading: creating }"
      @confirm="createSandbox"
    >
      <t-form :data="createForm" :rules="formRules" ref="formRef">
        <t-form-item label="镜像" name="image">
          <t-input v-model="createForm.image" placeholder="如：python:3.11-slim" />
        </t-form-item>
        <t-form-item label="CPU">
          <t-input v-model="createForm.cpu" placeholder="如：500m" />
        </t-form-item>
        <t-form-item label="内存">
          <t-input v-model="createForm.memory" placeholder="如：512Mi" />
        </t-form-item>
        <t-form-item label="TTL (秒)">
          <t-input-number v-model="createForm.ttl" :min="60" :max="86400" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { sandboxApi, type Sandbox, type CreateSandboxRequest } from '../api/sandbox'

const router = useRouter()
const sandboxes = ref<Sandbox[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const creating = ref(false)
const formRef = ref()

const createForm = ref<CreateSandboxRequest>({
  image: 'python:3.11-slim',
  cpu: '500m',
  memory: '512Mi',
  ttl: 3600,
})

const formRules = {
  image: [{ required: true, message: '请输入镜像名称' }],
}

const columns = [
  { colKey: 'id', title: 'ID', width: 120 },
  { colKey: 'image', title: '镜像', ellipsis: true },
  { colKey: 'cpu', title: 'CPU', width: 100 },
  { colKey: 'memory', title: '内存', width: 100 },
  { colKey: 'status', title: '状态', width: 100 },
  { colKey: 'created_at', title: '创建时间', width: 180 },
  { colKey: 'expires_at', title: '过期时间', width: 180 },
  { colKey: 'operation', title: '操作', width: 120 },
]

const getStatusTheme = (status: string) => {
  switch (status) {
    case 'running':
      return 'success'
    case 'pending':
      return 'warning'
    case 'failed':
      return 'danger'
    default:
      return 'default'
  }
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString()
}

const loadSandboxes = async () => {
  loading.value = true
  try {
    const res = await sandboxApi.list()
    sandboxes.value = res.data.items || []
  } catch (err: any) {
    MessagePlugin.error('加载失败: ' + (err.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const createSandbox = async () => {
  const valid = await formRef.value?.validate()
  if (valid !== true) return

  creating.value = true
  try {
    await sandboxApi.create(createForm.value)
    MessagePlugin.success('创建成功')
    showCreateDialog.value = false
    loadSandboxes()
  } catch (err: any) {
    MessagePlugin.error('创建失败: ' + (err.response?.data?.error || err.message))
  } finally {
    creating.value = false
  }
}

const deleteSandbox = async (id: string) => {
  try {
    await sandboxApi.delete(id)
    MessagePlugin.success('删除成功')
    loadSandboxes()
  } catch (err: any) {
    MessagePlugin.error('删除失败: ' + (err.response?.data?.error || err.message))
  }
}

const goToDetail = (id: string) => {
  router.push(`/sandboxes/${id}`)
}

let refreshInterval: number

onMounted(() => {
  loadSandboxes()
  refreshInterval = window.setInterval(loadSandboxes, 5000)
})

onUnmounted(() => {
  clearInterval(refreshInterval)
})
</script>

<style scoped>
.sandbox-list {
  padding: 24px;
}
</style>
