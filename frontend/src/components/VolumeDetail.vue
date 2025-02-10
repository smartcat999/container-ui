<template>
  <el-dialog
    v-model="visible"
    :title="'数据卷详情 - ' + (volumeInfo?.Name || '')"
    width="80%"
  >
    <el-descriptions v-if="volumeInfo" :column="2" border>
      <el-descriptions-item label="名称">{{ volumeInfo.Name }}</el-descriptions-item>
      <el-descriptions-item label="驱动类型">{{ volumeInfo.Driver }}</el-descriptions-item>
      <el-descriptions-item label="挂载点">{{ volumeInfo.Mountpoint }}</el-descriptions-item>
      <el-descriptions-item label="创建时间">{{ formatDate(volumeInfo.CreatedAt) }}</el-descriptions-item>
      <el-descriptions-item label="范围">{{ volumeInfo.Scope }}</el-descriptions-item>
      <el-descriptions-item label="状态">{{ volumeInfo.Status || '-' }}</el-descriptions-item>
      <el-descriptions-item label="标签" :span="2">
        <template v-if="Object.keys(volumeInfo.Labels || {}).length">
          <el-tag 
            v-for="(value, key) in volumeInfo.Labels" 
            :key="key"
            class="tag-item"
          >
            {{ key }}: {{ value }}
          </el-tag>
        </template>
        <template v-else>无标签</template>
      </el-descriptions-item>
      <el-descriptions-item label="选项" :span="2">
        <template v-if="Object.keys(volumeInfo.Options || {}).length">
          <div v-for="(value, key) in volumeInfo.Options" :key="key">
            {{ key }}: {{ value }}
          </div>
        </template>
        <template v-else>无选项</template>
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

const visible = ref(false)
const volumeInfo = ref<any>(null)

const formatDate = (timestamp: string) => {
  return new Date(timestamp).toLocaleString()
}

const show = async (name: string) => {
  try {
    visible.value = true
    const response = await dockerApi.getVolumeDetail(name)
    volumeInfo.value = response.data
  } catch (error) {
    ElMessage.error('获取数据卷详情失败')
    console.error('Error fetching volume detail:', error)
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