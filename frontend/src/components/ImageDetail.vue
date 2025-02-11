<template>
  <el-dialog
    v-model="visible"
    :title="'镜像详情 - ' + (imageInfo?.RepoTags?.[0] || imageInfo?.Id?.substring(7, 19))"
    width="80%"
  >
    <el-descriptions v-if="imageInfo" :column="2" border>
      <el-descriptions-item label="镜像ID">{{ formatId(imageInfo.Id) }}</el-descriptions-item>
      <el-descriptions-item label="创建时间">{{ formatDate(imageInfo.Created) }}</el-descriptions-item>
      <el-descriptions-item label="大小">{{ formatSize(imageInfo.Size) }}</el-descriptions-item>
      <el-descriptions-item label="虚拟大小">{{ formatSize(imageInfo.VirtualSize) }}</el-descriptions-item>
      <el-descriptions-item label="标签" :span="2">
        <el-tag v-for="tag in imageInfo.RepoTags" :key="tag" class="tag-item">
          {{ tag }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="架构">{{ imageInfo.Architecture }}</el-descriptions-item>
      <el-descriptions-item label="操作系统">{{ imageInfo.Os }}</el-descriptions-item>
      <el-descriptions-item label="Docker版本">{{ imageInfo.DockerVersion }}</el-descriptions-item>
      <el-descriptions-item label="作者">{{ imageInfo.Author || '-' }}</el-descriptions-item>
      <el-descriptions-item label="环境变量" :span="2" v-if="imageInfo.Config?.Env?.length">
        <el-tag v-for="env in imageInfo.Config.Env" :key="env" class="tag-item">
          {{ env }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="暴露端口" :span="2" v-if="imageInfo.Config?.ExposedPorts">
        <el-tag v-for="(_, port) in imageInfo.Config.ExposedPorts" :key="port" class="tag-item">
          {{ port }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="工作目录" v-if="imageInfo.Config?.WorkingDir">
        {{ imageInfo.Config.WorkingDir }}
      </el-descriptions-item>
      <el-descriptions-item label="默认命令" v-if="imageInfo.Config?.Cmd">
        {{ imageInfo.Config.Cmd.join(' ') }}
      </el-descriptions-item>
      <el-descriptions-item label="卷挂载点" :span="2" v-if="imageInfo.Config?.Volumes">
        <div v-for="(_, volume) in imageInfo.Config.Volumes" :key="volume">
          {{ volume }}
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
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { dockerApi } from '@/api/docker'
import { useContextStore } from '@/store/context'

const visible = ref(false)
const imageInfo = ref<any>(null)
const contextStore = useContextStore()

const formatId = (id: string) => {
  return id?.substring(7, 19) || ''
}

const formatDate = (timestamp: string) => {
  return new Date(timestamp).toLocaleString()
}

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

const show = async (id: string) => {
  try {
    visible.value = true
    const response = await dockerApi.getImageDetail(contextStore.getCurrentContext(), id)
    imageInfo.value = response.data
  } catch (error) {
    ElMessage.error('获取镜像详情失败')
    console.error('Error fetching image detail:', error)
  }
}

defineExpose({
  show
})
</script>

<style scoped>
.tag-item {
  margin-right: 8px;
  margin-bottom: 8px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style> 