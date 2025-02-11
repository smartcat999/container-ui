<template>
  <el-dialog
    v-model="visible"
    :title="'容器日志 - ' + containerName"
    width="80%"
    :destroy-on-close="true"
  >
    <div class="logs-container">
      <div class="logs-header">
        <el-switch
          v-model="autoScroll"
          active-text="自动滚动"
        />
        <el-button type="primary" @click="refreshLogs">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
      <div 
        ref="logsRef"
        class="logs-content"
        :class="{ 'auto-scroll': autoScroll }"
      >
        <pre v-if="logs">{{ logs }}</pre>
        <el-empty v-else description="暂无日志" />
      </div>
    </div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="visible = false">关闭</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import { useContextStore } from '@/store/context'

const visible = ref(false)
const logs = ref<string>('')
const autoScroll = ref(true)
const logsRef = ref<HTMLElement>()
const containerId = ref('')
const containerName = ref('')
const contextStore = useContextStore()

watch(() => visible.value, (newVal) => {
  if (!newVal) {
    logs.value = ''
  }
})

watch(() => logs.value, async () => {
  if (autoScroll.value) {
    await nextTick()
    if (logsRef.value) {
      logsRef.value.scrollTop = logsRef.value.scrollHeight
    }
  }
})

const refreshLogs = async () => {
  if (!containerId.value) return
  
  try {
    const response = await dockerApi.getContainerLogs(contextStore.getCurrentContext(), containerId.value)
    logs.value = response.data
  } catch (error) {
    ElMessage.error('获取容器日志失败')
    console.error('Error fetching container logs:', error)
  }
}

const show = (id: string, name: string) => {
  visible.value = true
  containerId.value = id
  containerName.value = name
  refreshLogs()
}

defineExpose({
  show
})
</script>

<style scoped>
.logs-container {
  height: 60vh;
  display: flex;
  flex-direction: column;
}

.logs-header {
  display: flex;
  justify-content: flex-end;
  gap: 16px;
  margin-bottom: 16px;
  align-items: center;
}

.logs-content {
  flex: 1;
  background-color: #1e1e1e;
  color: #fff;
  padding: 16px;
  overflow-y: auto;
  border-radius: 4px;
  font-family: monospace;
}

.logs-content pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style> 