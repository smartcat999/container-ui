<template>
  <el-dialog
    v-model="visible"
    :title="'容器详情 - ' + (containerInfo?.Name || '')"
    width="80%"
  >
    <el-descriptions v-if="containerInfo" :column="2" border>
      <el-descriptions-item label="容器ID">{{ containerInfo.Id }}</el-descriptions-item>
      <el-descriptions-item label="创建时间">{{ formatDate(containerInfo.Created) }}</el-descriptions-item>
      <el-descriptions-item label="状态">
        <el-tag :type="getStatusType(containerInfo.State.Status)">
          {{ containerInfo.State.Status }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="重启次数">{{ containerInfo.RestartCount }}</el-descriptions-item>
      <el-descriptions-item label="镜像">{{ containerInfo.Config.Image }}</el-descriptions-item>
      <el-descriptions-item label="工作目录">{{ containerInfo.Config.WorkingDir || '-' }}</el-descriptions-item>
      <el-descriptions-item label="环境变量" :span="2">
        <el-tag v-for="env in containerInfo.Config.Env" :key="env" class="env-tag">
          {{ env }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="端口映射" :span="2">
        <template v-if="Object.keys(containerInfo.NetworkSettings.Ports || {}).length">
          <div v-for="(port, key) in containerInfo.NetworkSettings.Ports" :key="key">
            {{ key }} -> {{ formatPorts(port) }}
          </div>
        </template>
        <template v-else>无端口映射</template>
      </el-descriptions-item>
      <el-descriptions-item label="数据卷" :span="2">
        <template v-if="containerInfo.Mounts.length">
          <div v-for="mount in containerInfo.Mounts" :key="mount.Source">
            {{ mount.Source }} -> {{ mount.Destination }}
            ({{ mount.Type }})
          </div>
        </template>
        <template v-else>无数据卷</template>
      </el-descriptions-item>
      <el-descriptions-item label="网络" :span="2">
        <div v-for="(network, name) in containerInfo.NetworkSettings.Networks" :key="name">
          {{ name }}: {{ network.IPAddress }}
        </div>
      </el-descriptions-item>
    </el-descriptions>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="visible = false">关闭</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, defineProps, defineExpose } from 'vue'
import { ElMessage } from 'element-plus'
import { dockerApi } from '@/api/docker'

const props = defineProps<{
  containerId?: string
}>()

const visible = ref(false)
const containerInfo = ref<any>(null)

const getStatusType = (status: string) => {
  switch (status) {
    case 'running':
      return 'success'
    case 'exited':
      return 'danger'
    case 'created':
      return 'info'
    default:
      return 'warning'
  }
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}

const formatPorts = (ports: any[]) => {
  if (!ports) return '未映射'
  return ports.map(p => `${p.HostIp || '0.0.0.0'}:${p.HostPort}`).join(', ')
}

const show = async (id: string) => {
  try {
    visible.value = true
    const response = await dockerApi.getContainerDetail(id)
    containerInfo.value = response.data
  } catch (error) {
    ElMessage.error('获取容器详情失败')
    console.error('Error fetching container detail:', error)
  }
}

defineExpose({
  show
})
</script>

<style scoped>
.env-tag {
  margin-right: 8px;
  margin-bottom: 8px;
}
</style> 