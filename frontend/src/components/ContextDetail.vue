<template>
  <el-dialog
    v-model="visible"
    :title="context?.name + ' 连接详情'"
    width="700px"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <el-descriptions :column="2" border>
      <el-descriptions-item label="连接名称" :span="2">{{ context?.name }}</el-descriptions-item>
      <el-descriptions-item label="连接类型">{{ context?.type.toUpperCase() }}</el-descriptions-item>
      <el-descriptions-item label="连接地址">{{ context?.host }}</el-descriptions-item>
      <el-descriptions-item label="连接状态" :span="2">
        <el-tag :type="serverInfo ? 'success' : 'danger'">
          {{ serverInfo ? '正常' : '异常' }}
        </el-tag>
      </el-descriptions-item>
    </el-descriptions>

    <div v-if="serverInfo" class="server-info">
      <el-divider>服务器信息</el-divider>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="Docker 版本">{{ serverInfo.Version }}</el-descriptions-item>
        <el-descriptions-item label="API 版本">{{ serverInfo.ApiVersion }}</el-descriptions-item>
        <el-descriptions-item label="操作系统">{{ serverInfo.OperatingSystem }}</el-descriptions-item>
        <el-descriptions-item label="系统架构">{{ serverInfo.Architecture }}</el-descriptions-item>
        <el-descriptions-item label="内核版本">{{ serverInfo.KernelVersion }}</el-descriptions-item>
        <el-descriptions-item label="CPU 核数">{{ serverInfo.NCPU }}</el-descriptions-item>
        <el-descriptions-item label="内存总量">{{ formatMemory(serverInfo.MemTotal) }}</el-descriptions-item>
        <el-descriptions-item label="存储驱动">{{ serverInfo.Driver }}</el-descriptions-item>
        <el-descriptions-item label="容器数量">{{ serverInfo.Containers }}</el-descriptions-item>
        <el-descriptions-item label="镜像数量">{{ serverInfo.Images }}</el-descriptions-item>
        <el-descriptions-item label="数据目录" :span="2">{{ serverInfo.DockerRootDir }}</el-descriptions-item>
      </el-descriptions>

      <el-divider>运行时信息</el-divider>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="运行时">{{ serverInfo.DefaultRuntime }}</el-descriptions-item>
        <el-descriptions-item label="Cgroup 驱动">{{ serverInfo.CgroupDriver }}</el-descriptions-item>
        <el-descriptions-item label="日志驱动">{{ serverInfo.LoggingDriver }}</el-descriptions-item>
        <el-descriptions-item label="Swarm 状态">{{ serverInfo.Swarm?.LocalNodeState || 'inactive' }}</el-descriptions-item>
      </el-descriptions>
    </div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
        <el-button type="primary" @click="handleRefresh">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import type { ContextConfig } from '@/api/docker'

interface Props {
  modelValue: boolean
  context?: ContextConfig
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: false
})

const emit = defineEmits(['update:modelValue'])

const visible = ref(props.modelValue)
const serverInfo = ref<any>(null)
const loading = ref(false)

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val && props.context) {
    loadServerInfo()
  }
})

watch(() => visible.value, (val) => {
  emit('update:modelValue', val)
})

const loadServerInfo = async () => {
  if (!props.context) return
  
  loading.value = true
  try {
    const response = await dockerApi.getServerInfo(props.context.name)
    serverInfo.value = response.data
  } catch (error: any) {
    ElMessage.error('获取服务器信息失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const formatMemory = (bytes: number): string => {
  if (!bytes) return '未知'
  const gb = bytes / (1024 * 1024 * 1024)
  return `${gb.toFixed(2)} GB`
}

const handleClose = () => {
  visible.value = false
}

const handleRefresh = () => {
  loadServerInfo()
}
</script>

<style scoped>
.server-info {
  margin-top: 20px;
}

.el-descriptions {
  margin: 16px 0;
}

:deep(.el-descriptions__cell) {
  min-width: 120px;
}
</style> 