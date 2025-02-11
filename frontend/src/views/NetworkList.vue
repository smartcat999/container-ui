<template>
  <div class="network-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>网络列表</span>
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
        <el-table-column prop="id" label="网络ID" width="120" />
        <el-table-column prop="name" label="名称" width="180" />
        <el-table-column prop="driver" label="驱动类型" width="120" />
        <el-table-column prop="scope" label="范围" width="120" />
        <el-table-column label="IPAM" width="200">
          <template #default="{ row }">
            <div v-for="config in row.ipam.config" :key="config.subnet">
              {{ config.subnet }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="scope">
            <el-button
              size="small"
              type="info"
              @click="showNetworkInfo(scope.row)"
            >
              详情
            </el-button>
            <el-popconfirm
              title="确定要删除这个网络吗？"
              @confirm="deleteNetwork(scope.row.id)"
            >
              <template #reference>
                <el-button
                  size="small"
                  type="danger"
                  :disabled="isSystemNetwork(scope.row.name)"
                >
                  删除
                </el-button>
              </template>
            </el-popconfirm>
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

    <!-- 网络详情对话框 -->
    <network-detail ref="networkDetailRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import type { Network } from '@/api/docker'
import NetworkDetail from '@/components/NetworkDetail.vue'
import { useContextStore } from '@/store/context'

const networks = ref<Network[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const networkDetailRef = ref()
const contextStore = useContextStore()

// 计算总数
const total = computed(() => networks.value.length)

// 计算当前页数据
const pageData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return networks.value.slice(start, end)
})

const formatDate = (timestamp: string) => {
  return new Date(timestamp).toLocaleString()
}

// 检查是否为系统网络（不允许删除）
const isSystemNetwork = (name: string) => {
  const systemNetworks = ['bridge', 'host', 'none']
  return systemNetworks.includes(name)
}

const refreshList = async () => {
  try {
    loading.value = true
    const response = await dockerApi.getNetworks(contextStore.getCurrentContext())
    networks.value = response.data
  } catch (error) {
    networks.value = [] // 清空列表
    ElMessage.error('获取网络列表失败')
    console.error('Error fetching networks:', error)
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

const showNetworkInfo = (network: Network) => {
  networkDetailRef.value?.show(network.id)
}

const deleteNetwork = async (id: string) => {
  try {
    loading.value = true
    await dockerApi.deleteNetwork(contextStore.getCurrentContext(), id)
    ElMessage.success('网络删除成功')
    refreshList()
  } catch (error) {
    ElMessage.error('网络删除失败')
    console.error('Error deleting network:', error)
  } finally {
    loading.value = false
  }
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
.network-list {
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

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style> 