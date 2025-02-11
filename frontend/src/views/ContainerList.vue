<template>
  <div class="container-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="search-filters">
            <el-input
              v-model="searchQuery"
              placeholder="搜索容器ID/名称/镜像"
              style="width: 300px; margin-right: 16px;"
              clearable
              @clear="handleSearch"
              @input="handleSearch"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
            <el-select
              v-model="stateFilter"
              placeholder="状态筛选"
              style="width: 140px; margin-right: 16px;"
              clearable
              @change="handleSearch"
            >
              <el-option
                v-for="state in stateOptions"
                :key="state.value"
                :label="state.label"
                :value="state.value"
              />
            </el-select>
          </div>
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
            <el-button-group>
              <el-button
                size="small"
                type="primary"
                :loading="scope.row.loading"
                :disabled="scope.row.state !== 'running'"
                @click="openTerminal(scope.row)"
              >
                <el-icon><Monitor /></el-icon>
                终端
              </el-button>
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
            </el-button-group>
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
    
    <!-- 添加终端组件 -->
    <container-terminal ref="terminalRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Monitor, Search } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import type { Container } from '@/api/docker'
import ContainerDetail from '@/components/ContainerDetail.vue'
import ContainerLogs from '@/components/ContainerLogs.vue'
import ContainerTerminal from '@/components/ContainerTerminal.vue'
import { useContextStore } from '@/store/context'

const containers = ref<Container[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const containerDetailRef = ref()
const containerLogsRef = ref()
const terminalRef = ref()

const searchQuery = ref('')
const stateFilter = ref('')

const stateOptions = [
  { label: '运行中', value: 'running' },
  { label: '已停止', value: 'exited' },
  { label: '已创建', value: 'created' }
]

const contextStore = useContextStore()

// 过滤后的数据
const filteredContainers = computed(() => {
  return containers.value.filter(container => {
    // 状态筛选
    if (stateFilter.value && container.state !== stateFilter.value) {
      return false
    }
    
    // 搜索查询
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      return (
        container.id.toLowerCase().includes(query) ||
        container.name.toLowerCase().includes(query) ||
        container.image.toLowerCase().includes(query)
      )
    }
    
    return true
  })
})

// 更新分页数据计算
const total = computed(() => filteredContainers.value.length)

const pageData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredContainers.value.slice(start, end)
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
    const response = await dockerApi.getContainers(contextStore.getCurrentContext())
    containers.value = response.data
  } catch (error) {
    containers.value = [] // 清空列表
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
    await dockerApi.startContainer(contextStore.getCurrentContext(), id)
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
    await dockerApi.stopContainer(contextStore.getCurrentContext(), id)
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
    await dockerApi.deleteContainer(contextStore.getCurrentContext(), container.id, isRunning)
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

const setContainerLoading = (containerId: string, loading: boolean) => {
  const container = containers.value.find(c => c.id === containerId)
  if (container) {
    container.loading = loading
  }
}

const openTerminal = async (container: Container) => {
  try {
    setContainerLoading(container.id, true)
    terminalRef.value?.show(container.id)
  } finally {
    setContainerLoading(container.id, false)
  }
}

// 搜索处理函数
const handleSearch = () => {
  currentPage.value = 1 // 重置到第一页
}

const handleContextChange = () => {
  refreshList()
}

onMounted(() => {
  refreshList()
  window.addEventListener('context-changed', handleContextChange)
})

onBeforeUnmount(() => {
  window.removeEventListener('context-changed', handleContextChange)
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

.search-filters {
  display: flex;
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