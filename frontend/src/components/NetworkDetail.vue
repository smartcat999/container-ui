<template>
  <el-dialog
    v-model="visible"
    :title="'网络详情 - ' + (networkInfo?.Name || '')"
    width="80%"
  >
    <el-descriptions v-if="networkInfo" :column="2" border>
      <el-descriptions-item label="网络ID">{{ networkInfo.Id }}</el-descriptions-item>
      <el-descriptions-item label="名称">{{ networkInfo.Name }}</el-descriptions-item>
      <el-descriptions-item label="驱动类型">{{ networkInfo.Driver }}</el-descriptions-item>
      <el-descriptions-item label="范围">{{ networkInfo.Scope }}</el-descriptions-item>
      <el-descriptions-item label="启用IPv6">{{ networkInfo.EnableIPv6 ? '是' : '否' }}</el-descriptions-item>
      <el-descriptions-item label="内部网络">{{ networkInfo.Internal ? '是' : '否' }}</el-descriptions-item>
      <el-descriptions-item label="IPAM" :span="2">
        <div v-for="config in networkInfo.IPAM?.Config" :key="config.Subnet">
          子网: {{ config.Subnet }}<br>
          网关: {{ config.Gateway }}
        </div>
      </el-descriptions-item>
      <el-descriptions-item label="已连接容器" :span="2">
        <template v-if="Object.keys(networkInfo.Containers || {}).length">
          <div v-for="(container, id) in networkInfo.Containers" :key="id">
            {{ container.Name }} ({{ container.IPv4Address }})
          </div>
        </template>
        <template v-else>无已连接容器</template>
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
const networkInfo = ref<any>(null)

const show = async (id: string) => {
  try {
    visible.value = true
    const response = await dockerApi.getNetworkDetail(id)
    networkInfo.value = response.data
  } catch (error) {
    ElMessage.error('获取网络详情失败')
    console.error('Error fetching network detail:', error)
  }
}

defineExpose({
  show
})
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style> 