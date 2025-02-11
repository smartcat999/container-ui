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
      <el-descriptions-item label="运行状态" :span="2">
        <div>启动时间：{{ formatDate(containerInfo.State.StartedAt) }}</div>
        <div v-if="containerInfo.State.Status === 'exited'">
          退出时间：{{ formatDate(containerInfo.State.FinishedAt) }}
          <br>
          退出码：{{ containerInfo.State.ExitCode }}
          <br>
          错误信息：{{ containerInfo.State.Error || '无' }}
        </div>
        <div v-if="containerInfo.State.Status === 'running'">
          进程ID：{{ containerInfo.State.Pid }}
        </div>
      </el-descriptions-item>
      <el-descriptions-item label="镜像">{{ containerInfo.Config.Image }}</el-descriptions-item>
      <el-descriptions-item label="主机名">{{ containerInfo.Config.Hostname }}</el-descriptions-item>
      <el-descriptions-item label="工作目录">{{ containerInfo.Config.WorkingDir || '-' }}</el-descriptions-item>
      <el-descriptions-item label="用户">{{ containerInfo.Config.User || 'root' }}</el-descriptions-item>
      <el-descriptions-item label="启动命令" :span="2">
        <div class="command-container">
          <div class="command-line">
            <div class="command-path">
              <el-tag type="info" class="command-tag">{{ containerInfo.Path || '默认启动命令' }}</el-tag>
            </div>
            <div class="command-args">
              <template v-if="containerInfo.Args && containerInfo.Args.length">
                <el-tag 
                  v-for="arg in containerInfo.Args" 
                  :key="arg" 
                  class="arg-tag" 
                  type="warning"
                >
                  {{ arg }}
                </el-tag>
              </template>
            </div>
          </div>
          <div class="command-text">
            完整命令：{{ [containerInfo.Path, ...(containerInfo.Args || [])].join(' ') }}
          </div>
        </div>
      </el-descriptions-item>
      <el-descriptions-item label="环境变量" :span="2">
        <el-tag v-for="env in containerInfo.Config.Env" 
                :key="env" 
                class="env-tag"
                type="success">
          {{ env }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="端口映射" :span="2">
        <template v-if="Object.keys(containerInfo.NetworkSettings.Ports || {}).length">
          <div v-for="(port, key) in containerInfo.NetworkSettings.Ports" :key="key">
            <el-tag type="info">{{ key }}</el-tag> → 
            <el-tag type="success">{{ formatPorts(port) }}</el-tag>
          </div>
        </template>
        <template v-else>无端口映射</template>
      </el-descriptions-item>
      <el-descriptions-item label="数据卷" :span="2">
        <template v-if="containerInfo.Mounts && containerInfo.Mounts.length">
          <div class="volumes-container">
            <div v-for="mount in containerInfo.Mounts" :key="mount.Source" class="volume-item">
              <div class="volume-mapping">
                <el-tag size="small" type="info" class="volume-tag">{{ mount.Type }}</el-tag>
                <el-tag size="small" type="primary" class="volume-tag">{{ mount.Source }}</el-tag>
                <el-icon class="volume-arrow"><ArrowRight /></el-icon>
                <el-tag size="small" type="success" class="volume-tag">{{ mount.Destination }}</el-tag>
                <el-tag size="small" type="warning" class="volume-tag">{{ mount.Mode || 'rw' }}</el-tag>
              </div>
              <div class="volume-details">
                <span class="volume-detail-item">类型: {{ mount.Type }}</span>
                <span class="volume-detail-item">读写: {{ mount.RW ? '读写' : '只读' }}</span>
                <span v-if="mount.Propagation" class="volume-detail-item">
                  传播: {{ mount.Propagation }}
                </span>
              </div>
            </div>
          </div>
        </template>
        <template v-else>
          <el-empty description="无数据卷" :image-size="60" />
        </template>
      </el-descriptions-item>
      <el-descriptions-item label="网络" :span="2">
        <div v-for="(network, name) in containerInfo.NetworkSettings.Networks" :key="name">
          <el-tag type="info">{{ name }}</el-tag>
          <div class="network-details">
            <div>IP地址：{{ network.IPAddress }}</div>
            <div>网关：{{ network.Gateway }}</div>
            <div>MAC地址：{{ network.MacAddress }}</div>
          </div>
        </div>
      </el-descriptions-item>
      <el-descriptions-item label="资源限制" :span="2">
        <div>内存限制：{{ formatBytes(containerInfo.HostConfig.Memory) }}</div>
        <div>CPU限制：{{ containerInfo.HostConfig.CpuShares || '无限制' }}</div>
        <div>重启策略：{{ containerInfo.HostConfig.RestartPolicy.Name }}</div>
      </el-descriptions-item>
      <el-descriptions-item label="健康检查" :span="2" v-if="containerInfo.State.Health">
        <div>状态：{{ containerInfo.State.Health.Status }}</div>
        <div v-if="containerInfo.State.Health.Log">
          最近检查：
          <div v-for="(log, index) in containerInfo.State.Health.Log.slice(-3)" :key="index">
            {{ formatDate(log.Start) }} - {{ log.ExitCode === 0 ? '成功' : '失败' }}
          </div>
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
import { ArrowRight } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import { useContextStore } from '@/store/context'

const props = defineProps<{
  containerId?: string
}>()

const visible = ref(false)
const containerInfo = ref<any>(null)
const contextStore = useContextStore()

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
  if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return '未设置'
  return new Date(dateStr).toLocaleString()
}

const formatPorts = (ports: any[]) => {
  if (!ports) return '未映射'
  return ports.map(p => `${p.HostIp || '0.0.0.0'}:${p.HostPort}`).join(', ')
}

const formatBytes = (bytes: number) => {
  if (!bytes || bytes === 0) return '无限制'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex++
  }
  return `${value.toFixed(2)} ${units[unitIndex]}`
}

const show = async (id: string) => {
  try {
    visible.value = true
    const response = await dockerApi.getContainerDetail(contextStore.getCurrentContext(), id)
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
.env-tag, .arg-tag {
  margin-right: 8px;
  margin-bottom: 8px;
}

.command-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.command-line {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.command-path {
  flex-shrink: 0;
}

.command-args {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.command-tag,
.arg-tag {
  font-family: monospace;
  height: 32px;
  line-height: 32px;
  display: inline-flex;
  align-items: center;
}

.command-text {
  color: var(--el-text-color-secondary);
  font-family: monospace;
  padding: 8px;
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
  word-break: break-all;
  margin-top: 4px;
}

.volumes-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.volume-item {
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
  padding: 8px;
}

.volume-mapping {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.volume-arrow {
  color: var(--el-text-color-secondary);
}

.volume-tag {
  white-space: normal;
  height: auto;
  padding: 4px 8px;
  line-height: 1.4;
}

.volume-details {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-secondary);
  font-size: 0.9em;
  display: flex;
  gap: 16px;
}

.volume-detail-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.network-details {
  margin-left: 16px;
  margin-top: 4px;
  margin-bottom: 8px;
  color: var(--el-text-color-secondary);
}

:deep(.el-descriptions__cell) {
  padding: 16px;
}

:deep(.el-empty) {
  padding: 20px 0;
}
</style> 