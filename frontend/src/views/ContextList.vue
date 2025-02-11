<template>
  <div class="context-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Docker 连接管理</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            新增连接
          </el-button>
        </div>
      </template>

      <el-table :data="contextList" style="width: 100%">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="type" label="类型">
          <template #default="{ row }">
            {{ row.type.toUpperCase() }}
          </template>
        </el-table-column>
        <el-table-column prop="host" label="连接地址" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.current" type="success">当前</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button-group>
              <el-button
                type="primary"
                :disabled="row.current"
                @click="handleUseContext(row.name)"
              >
                使用
              </el-button>
              <el-button
                type="primary"
                @click="showEditDialog(row)"
              >
                编辑
              </el-button>
              <el-button
                type="danger"
                :disabled="row.current"
                @click="handleDeleteContext(row.name)"
              >
                删除
              </el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增/编辑连接对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEditing ? '编辑连接' : '新增连接'"
      width="500px"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="连接名称" prop="name" v-if="!isEditing">
          <el-input v-model="form.name" placeholder="请输入名称" />
        </el-form-item>
        
        <el-form-item label="连接方式" prop="type">
          <el-radio-group v-model="form.type">
            <el-radio :value="'tcp'">TCP</el-radio>
            <el-radio :value="'socket'">Socket</el-radio>
          </el-radio-group>
        </el-form-item>

        <template v-if="form.type === 'tcp'">
          <el-form-item label="主机地址" prop="host">
            <el-input v-model="form.host" placeholder="例如: 192.168.1.100" />
          </el-form-item>
          <el-form-item label="端口">
            <el-input-number v-model="form.port" :min="1" :max="65535" :step="1" />
          </el-form-item>
        </template>

        <template v-if="form.type === 'socket'">
          <el-form-item label="Socket 路径" prop="socketPath">
            <el-input 
              v-model="form.socketPath" 
              placeholder="例如: /var/run/docker.sock"
            >
              <template #append>
                <el-button @click="useDefaultSocket">默认</el-button>
              </template>
            </el-input>
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveContext" :loading="saving">
            确认
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import type { ContextConfig, ContextType } from '@/api/docker'

const contextList = ref<ContextConfig[]>([])
const dialogVisible = ref(false)
const isEditing = ref(false)
const saving = ref(false)

const form = ref({
  name: '',
  type: 'tcp' as ContextType,
  host: '',
  port: 2375,
  socketPath: '/var/run/docker.sock',
  current: false
})

const rules = {
  name: [{ required: true, message: '请输入连接名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择连接方式', trigger: 'change' }],
  host: [{ 
    required: true, 
    message: '请输入主机地址', 
    trigger: 'blur',
    validator: (rule: any, value: string, callback: Function) => {
      if (form.value.type === 'tcp' && !value) {
        callback(new Error('请输入主机地址'))
      } else {
        callback()
      }
    }
  }],
  socketPath: [{
    required: true,
    message: '请输入 Socket 路径',
    trigger: 'blur',
    validator: (rule: any, value: string, callback: Function) => {
      if (form.value.type === 'socket' && !value) {
        callback(new Error('请输入 Socket 路径'))
      } else {
        callback()
      }
    }
  }]
}

const loadContexts = async () => {
  try {
    const response = await dockerApi.getContexts()
    contextList.value = response.data || []
  } catch (error) {
    ElMessage.error('加载连接列表失败')
    console.error('Error loading contexts:', error)
  }
}

const showCreateDialog = () => {
  isEditing.value = false
  form.value = {
    name: '',
    type: 'tcp',
    host: '',
    port: 2375,
    socketPath: '/var/run/docker.sock',
    current: false
  }
  dialogVisible.value = true
}

const showEditDialog = (context: ContextConfig) => {
  isEditing.value = true
  
  // 重置表单
  form.value = {
    name: context.name,
    type: context.type, // 先设置类型，确保表单显示正确的字段
    host: '',
    port: 2375,
    socketPath: '/var/run/docker.sock',
    current: context.current
  }

  // 根据类型回填对应字段
  if (context.type === 'tcp') {
    const match = context.host.match(/^tcp:\/\/([^:]+):(\d+)$/)
    if (match) {
      form.value.host = match[1]
      form.value.port = parseInt(match[2])
    }
  } else {
    form.value.socketPath = context.host.replace(/^unix:\/\//, '')
  }

  // 确保表单重新渲染
  nextTick(() => {
    dialogVisible.value = true
  })
}

const buildDockerHost = (formData: typeof form.value): string => {
  if (formData.type === 'tcp') {
    const host = formData.host || 'localhost'
    const port = formData.port || 2375
    return `tcp://${host}:${port}`
  } else {
    const socketPath = formData.socketPath || '/var/run/docker.sock'
    return socketPath.startsWith('unix://') ? socketPath : `unix://${socketPath}`
  }
}

const handleSaveContext = async () => {
  saving.value = true
  try {
    const config: ContextConfig = {
      name: form.value.name,
      type: form.value.type,
      host: buildDockerHost(form.value),
      current: form.value.current
    }

    if (isEditing.value) {
      await dockerApi.updateContextConfig(form.value.name, config)
      ElMessage.success('连接更新成功')
    } else {
      await dockerApi.createContext(config)
      ElMessage.success('连接创建成功')
    }
    
    dialogVisible.value = false
    await loadContexts()
  } catch (error) {
    ElMessage.error(isEditing.value ? '更新连接失败' : '创建连接失败')
    console.error('Error saving context:', error)
  } finally {
    saving.value = false
  }
}

const handleUseContext = async (name: string) => {
  try {
    await dockerApi.switchContext(name)
    ElMessage.success('切换连接成功')
    await loadContexts()
    window.dispatchEvent(new CustomEvent('context-changed'))
  } catch (error) {
    ElMessage.error('切换连接失败')
    console.error('Error switching context:', error)
  }
}

const handleDeleteContext = (name: string) => {
  ElMessageBox.confirm(
    `确定要删除连接 "${name}" 吗？`,
    '删除连接',
    {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      type: 'warning',
    }
  )
    .then(async () => {
      try {
        await dockerApi.deleteContext(name)
        ElMessage.success('删除连接成功')
        loadContexts()
      } catch (error: any) {
        if (error.response?.data?.error?.includes('cannot delete current context')) {
          ElMessage.error('无法删除当前使用的连接')
        } else {
          ElMessage.error('删除连接失败')
        }
        console.error('Error deleting context:', error)
      }
    })
    .catch(() => {
      // 用户取消删除，不做任何操作
    })
}

const useDefaultSocket = () => {
  form.value.socketPath = '/var/run/docker.sock'
}

onMounted(() => {
  loadContexts()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.el-button-group {
  .el-button {
    margin-left: -1px;
  }
}
</style> 