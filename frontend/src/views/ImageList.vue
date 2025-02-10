<template>
  <div class="image-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>镜像列表</span>
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
        <el-table-column prop="id" label="镜像ID" width="120" />
        <el-table-column prop="repository" label="仓库名称" width="200" />
        <el-table-column prop="tag" label="标签" width="120" />
        <el-table-column label="大小" width="120">
          <template #default="{ row }">
            {{ formatSize(row.size) }}
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
              type="primary"
              @click="createContainer(scope.row)"
            >
              创建容器
            </el-button>
            <el-button
              size="small"
              type="info"
              @click="showImageInfo(scope.row)"
            >
              详情
            </el-button>
            <el-popconfirm
              title="确定要删除这个镜像吗？"
              @confirm="deleteImage(scope.row.id)"
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

    <!-- 创建容器对话框 -->
    <el-dialog v-model="createDialogVisible" title="创建容器" width="500px">
      <el-form :model="containerForm" label-width="120px">
        <el-form-item label="容器名称">
          <el-input v-model="containerForm.name" placeholder="请输入容器名称" />
        </el-form-item>
        <el-form-item label="端口映射">
          <el-input v-model="containerForm.ports" placeholder="例如: 80:80,443:443" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmCreate">
            确认创建
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 添加镜像详情对话框 -->
    <image-detail ref="imageDetailRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import type { Image } from '@/api/docker'
import ImageDetail from '@/components/ImageDetail.vue'

const images = ref<Image[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const createDialogVisible = ref(false)
const containerForm = ref({
  name: '',
  ports: '',
  imageId: ''
})
const imageDetailRef = ref()

// 计算总数
const total = computed(() => images.value.length)

// 计算当前页数据
const pageData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return images.value.slice(start, end)
})

const formatSize = (size: number) => {
  const units = ['B', 'KB', 'MB', 'GB']
  let index = 0
  let formattedSize = size

  while (formattedSize >= 1024 && index < units.length - 1) {
    formattedSize /= 1024
    index++
  }

  return `${formattedSize.toFixed(2)} ${units[index]}`
}

const formatDate = (timestamp: number) => {
  return new Date(timestamp * 1000).toLocaleString()
}

const refreshList = async () => {
  try {
    loading.value = true
    const response = await dockerApi.getImages()
    images.value = response.data
  } catch (error) {
    ElMessage.error('获取镜像列表失败')
    console.error('Error fetching images:', error)
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

const createContainer = (image: Image) => {
  containerForm.value.imageId = image.id
  containerForm.value.name = ''
  containerForm.value.ports = ''
  createDialogVisible.value = true
}

const confirmCreate = async () => {
  try {
    loading.value = true
    await dockerApi.createContainer({
      imageId: containerForm.value.imageId,
      name: containerForm.value.name,
      ports: containerForm.value.ports.split(',').map(p => {
        const [host, container] = p.split(':')
        return { host, container }
      })
    })
    ElMessage.success('容器创建成功')
    createDialogVisible.value = false
  } catch (error) {
    ElMessage.error('容器创建失败')
    console.error('Error creating container:', error)
  } finally {
    loading.value = false
  }
}

const deleteImage = async (imageId: string) => {
  try {
    loading.value = true
    await dockerApi.deleteImage(imageId)
    ElMessage.success('镜像删除成功')
    refreshList()
  } catch (error) {
    ElMessage.error('镜像删除失败')
    console.error('Error deleting image:', error)
  } finally {
    loading.value = false
  }
}

const showImageInfo = (image: Image) => {
  imageDetailRef.value?.show(image.id)
}

onMounted(() => {
  refreshList()
})
</script>

<style scoped>
.image-list {
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

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style> 