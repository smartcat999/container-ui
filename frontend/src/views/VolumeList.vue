<template>
  <div class="volume-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>数据卷列表</span>
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
        <el-table-column prop="name" label="名称" width="200" />
        <el-table-column prop="driver" label="驱动类型" width="120" />
        <el-table-column prop="mountpoint" label="挂载点" width="300" show-overflow-tooltip />
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
              @click="showVolumeInfo(scope.row)"
            >
              详情
            </el-button>
            <el-popconfirm
              title="确定要删除这个数据卷吗？"
              @confirm="deleteVolume(scope.row.name)"
            >
              <template #reference>
                <el-button
                  size="small"
                  type="danger"
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

    <!-- 数据卷详情对话框 -->
    <volume-detail ref="volumeDetailRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import type { Volume } from '@/api/docker'
import VolumeDetail from '@/components/VolumeDetail.vue'

const volumes = ref<Volume[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const volumeDetailRef = ref()

// 计算总数
const total = computed(() => volumes.value.length)

// 计算当前页数据
const pageData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return volumes.value.slice(start, end)
})

const formatDate = (timestamp: string) => {
  return new Date(timestamp).toLocaleString()
}

const refreshList = async () => {
  try {
    loading.value = true
    const response = await dockerApi.getVolumes()
    volumes.value = response.data
  } catch (error) {
    ElMessage.error('获取数据卷列表失败')
    console.error('Error fetching volumes:', error)
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

const showVolumeInfo = (volume: Volume) => {
  volumeDetailRef.value?.show(volume.name)
}

const deleteVolume = async (name: string) => {
  try {
    loading.value = true
    await dockerApi.deleteVolume(name)
    ElMessage.success('数据卷删除成功')
    refreshList()
  } catch (error) {
    ElMessage.error('数据卷删除失败')
    console.error('Error deleting volume:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refreshList()
})
</script>

<style scoped>
.volume-list {
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