<template>
  <div class="container-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>容器列表</span>
          <el-button type="primary" @click="refreshList">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </template>
      
      <el-table 
        :data="pageData" 
        style="width: 100%"
        v-loading="loading"
      >
        <el-table-column prop="id" label="容器ID" width="120" />
        <el-table-column prop="name" label="名称" width="180" />
        <el-table-column prop="image" label="镜像" width="200" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.state)">
              {{ row.state }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="端口映射" width="200">
          <template #default="{ row }">
            <div v-for="port in row.ports" :key="`${port.privatePort}-${port.publicPort}`">
              {{ port.publicPort ? `${port.publicPort}:${port.privatePort}` : port.privatePort }} 
              ({{ port.type }})
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="详细状态" width="200" />
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="scope">
            <el-button
              size="small"
              :type="scope.row.state === 'running' ? 'danger' : 'success'"
              @click="scope.row.state === 'running' ? stopContainer(scope.row.id) : startContainer(scope.row.id)"
            >
              {{ scope.row.state === 'running' ? '停止' : '启动' }}
            </el-button>
            <el-button
              size="small"
              type="info"
              @click="showContainerInfo(scope.row)"
            >
              详情
            </el-button>
            <el-button
              size="small"
              type="warning"
              @click="showContainerLogs(scope.row)"
              :disabled="scope.row.state !== 'running'"
            >
              日志
            </el-button>
            <el-button
              size="small"
              type="danger"
              @click="deleteContainer(scope.row)"
              :loading="scope.row.loading"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页组件 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
    
    <!-- 容器详情对话框 -->
    <container-detail ref="containerDetailRef" />
    
    <!-- 添加容器日志对话框 -->
    <container-logs ref="containerLogsRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import type { Container } from '@/api/docker'
import ContainerDetail from '@/components/ContainerDetail.vue'
import ContainerLogs from '@/components/ContainerLogs.vue'

const containers = ref<Container[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const containerDetailRef = ref()
const containerLogsRef = ref()

// 计算总数
const total = computed(() => containers.value.length)

// 计算当前页数据
const pageData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return containers.value.slice(start, end)
})

const getStatusType = (state: string) => {
  switch (state) {
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

const formatDate = (timestamp: number) => {
  return new Date(timestamp * 1000).toLocaleString()
}

const refreshList = async () => {
  try {
    loading.value = true
    const response = await dockerApi.getContainers()
    containers.value = response.data
  } catch (error) {
    ElMessage.error('获取容器列表失败')
    console.error('Error fetching containers:', error)
  } finally {
    loading.value = false
  }
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  currentPage.value = 1 // 重置到第一页
}

const handleCurrentChange = (val: number) => {
  currentPage.value = val
}

const startContainer = async (id: string) => {
  try {
    loading.value = true
    await dockerApi.startContainer(id)
    ElMessage.success('容器启动成功')
    refreshList()
  } catch (error) {
    ElMessage.error('容器启动失败')
    console.error('Error starting container:', error)
  }
}

const stopContainer = async (id: string) => {
  try {
    loading.value = true
    await dockerApi.stopContainer(id)
    ElMessage.success('容器停止成功')
    refreshList()
  } catch (error) {
    ElMessage.error('容器停止失败')
    console.error('Error stopping container:', error)
  }
}

const showContainerInfo = (container: Container) => {
  containerDetailRef.value?.show(container.id)
}

const showContainerLogs = (container: Container) => {
  containerLogsRef.value?.show(container.id, container.name)
}

const deleteContainer = async (container: Container) => {
  const isRunning = container.state === 'running'
  
  try {
    await ElMessageBox.confirm(
      isRunning 
        ? '容器正在运行中，是否强制删除？'
        : '确定要删除该容器吗？',
      '删除容器',
      {
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    container.loading = true
    await dockerApi.deleteContainer(container.id, isRunning)
    ElMessage.success('删除容器成功')
    refreshList()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除容器失败')
      console.error('Error deleting container:', error)
    }
  } finally {
    container.loading = false
  }
}

onMounted(() => {
  refreshList()
})
</script>

<style scoped>
.container-list {
  height: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.el-button {
  margin-left: 8px;
}

.el-tag {
  text-transform: capitalize;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>