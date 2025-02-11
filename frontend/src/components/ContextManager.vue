<template>
  <el-dropdown trigger="click" @command="handleContextCommand">
    <span class="el-dropdown-link">
      <el-button type="primary" class="context-button" :title="currentContext?.name">
        <template #icon>
          <el-icon><Connection /></el-icon>
        </template>
        {{ currentContext?.name || '未连接' }}
        <el-icon class="el-icon--right"><arrow-down /></el-icon>
      </el-button>
    </span>
    <template #dropdown>
      <el-dropdown-menu>
        <template v-if="contexts?.length">
          <el-dropdown-item
            v-for="context in contexts"
            :key="context.name"
            :class="{ 'is-active': context.current }"
          >
            <span class="context-item">
              <span class="context-label" @click="handleContextCommand({ type: 'switch', name: context.name })">
                <el-icon><Connection /></el-icon>
                {{ context.name }}
                <el-tag v-if="context.current" size="small" type="success">当前</el-tag>
              </span>
              <span class="context-actions">
                <el-tooltip content="编辑" placement="top">
                  <span class="action-wrapper" @click.stop="showEditDialog(context)">
                    <el-icon class="action-icon"><Edit /></el-icon>
                  </span>
                </el-tooltip>
                <el-tooltip v-if="!context.current" content="删除" placement="top">
                  <span class="action-wrapper" @click.stop="confirmDelete(context.name)">
                    <el-icon class="action-icon"><Delete /></el-icon>
                  </span>
                </el-tooltip>
              </span>
            </span>
          </el-dropdown-item>
        </template>
        <el-dropdown-item v-else class="empty-item">
          <span class="empty-wrapper">
            <el-icon><Connection /></el-icon>
            <span>未添加连接</span>
          </span>
        </el-dropdown-item>
        <el-dropdown-item divided>
          <span class="add-item" @click="showAddContextDialog">
            <el-icon><Plus /></el-icon>
            <span>新增连接</span>
          </span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>

  <!-- 新增/编辑连接对话框 -->
  <el-dialog
    v-model="dialogVisible"
    :title="isEditing ? '编辑连接' : '新增连接'"
    width="500px"
  >
    <el-form
      ref="formRef"
      :model="contextForm"
      :rules="rules"
      label-width="120px"
    >
      <el-form-item label="连接名称" prop="name" v-if="!isEditing">
        <el-input v-model="contextForm.name" placeholder="请输入名称" />
      </el-form-item>
      
      <el-form-item label="连接方式" prop="type">
        <el-radio-group v-model="contextForm.type">
          <el-radio :value="'tcp'">TCP</el-radio>
          <el-radio :value="'socket'">Socket</el-radio>
        </el-radio-group>
      </el-form-item>

      <template v-if="contextForm.type === 'tcp'">
        <el-form-item label="主机地址" prop="host">
          <el-input v-model="contextForm.host" placeholder="例如: 192.168.1.100" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="contextForm.port" :min="1" :max="65535" :step="1" :default-value="2375" />
        </el-form-item>
      </template>

      <template v-if="contextForm.type === 'socket'">
        <el-form-item label="Socket 路径" prop="socketPath">
          <el-input 
            v-model="contextForm.socketPath" 
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
        <el-button type="primary" @click="handleSaveContext">确认</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, Plus, Edit, Delete, Connection } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import type { ContextConfig } from '@/api/docker'

interface ContextCommand {
  type: 'switch';
  name: string;
}

const currentContext = ref<ContextConfig | null>(null)
const contexts = ref<ContextConfig[]>([])
const dialogVisible = ref(false)
const isEditing = ref(false)
const contextForm = ref({
  name: '',
  type: 'tcp' as const,
  host: '',
  port: 2375,
  socketPath: '/var/run/docker.sock',
  current: false
})

const loadContexts = async () => {
  try {
    const response = await dockerApi.getContexts()
    contexts.value = response.data || []
    
    // 找到当前使用的上下文
    const currentCtx = contexts.value.find(ctx => ctx.current)
    if (currentCtx) {
      currentContext.value = currentCtx
    } else {
      currentContext.value = null
    }
  } catch (error) {
    ElMessage.error('加载 Docker Context 失败')
    console.error('Error loading contexts:', error)
  }
}

const handleContextCommand = async (command: { type: 'switch', name: string }) => {
  if (command.type === 'switch') {
    try {
      await dockerApi.switchContext(command.name)
      await loadContexts() // 重新加载上下文列表
      ElMessage.success('切换 Context 成功')
      window.dispatchEvent(new CustomEvent('context-changed'))
    } catch (error) {
      ElMessage.error('切换 Context 失败')
      console.error('Error switching context:', error)
    }
  }
}

const showAddContextDialog = () => {
  isEditing.value = false
  contextForm.value = {
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
  contextForm.value = {
    name: '',
    type: 'tcp',
    host: '',
    port: 2375,
    socketPath: '/var/run/docker.sock',
    current: false
  }

  // 根据类型解析并回填表单
  if (context.type === 'tcp') {
    // 解析 tcp://host:port 格式
    const match = context.host.match(/^tcp:\/\/([^:]+):(\d+)$/)
    if (match) {
      contextForm.value = {
        ...contextForm.value,
        name: context.name,
        type: 'tcp',
        host: match[1],
        port: parseInt(match[2]),
        current: context.current
      }
    }
  } else {
    // 解析 unix:// 格式
    contextForm.value = {
      ...contextForm.value,
      name: context.name,
      type: 'socket',
      socketPath: context.host.replace(/^unix:\/\//, ''),
      current: context.current
    }
  }
  
  dialogVisible.value = true
}

const confirmDelete = (contextName: string) => {
  ElMessageBox.confirm(
    `确定要删除 Context "${contextName}" 吗？`,
    '删除 Context',
    {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      type: 'warning',
    }
  )
    .then(async () => {
      try {
        await dockerApi.deleteContext(contextName)
        ElMessage.success('删除 Context 成功')
        loadContexts() // 重新加载上下文列表
      } catch (error: any) {
        if (error.response?.data?.error?.includes('cannot delete current context')) {
          ElMessage.error('无法删除当前使用的上下文')
        } else {
          ElMessage.error('删除 Context 失败')
        }
        console.error('Error deleting context:', error)
      }
    })
    .catch(() => {
      // 用户取消删除，不做任何操作
    })
}

const useDefaultSocket = () => {
  contextForm.value.socketPath = '/var/run/docker.sock'
}

const buildDockerHost = (form: typeof contextForm.value): string => {
  if (form.type === 'tcp') {
    const host = form.host || 'localhost'
    const port = form.port || 2375
    return `tcp://${host}:${port}`
  } else {
    const socketPath = form.socketPath || '/var/run/docker.sock'
    return socketPath.startsWith('unix://') ? socketPath : `unix://${socketPath}`
  }
}

const parseDockerHost = (host: string) => {
  if (host.startsWith('tcp://')) {
    const url = new URL(host)
    contextForm.value.type = 'tcp'
    contextForm.value.host = url.hostname
    contextForm.value.port = parseInt(url.port) || 2375
  } else if (host.startsWith('unix://')) {
    contextForm.value.type = 'socket'
    contextForm.value.socketPath = host.replace('unix://', '')
  }
}

const handleSaveContext = async () => {
  if (isEditing.value) {
    try {
      const config: ContextConfig = {
        name: contextForm.value.name,
        type: contextForm.value.type,
        host: buildDockerHost(contextForm.value),
        current: contextForm.value.current
      }
      await dockerApi.updateContextConfig(contextForm.value.name, config)
      ElMessage.success('连接更新成功')
      dialogVisible.value = false
      await loadContexts()
    } catch (error) {
      ElMessage.error('更新连接失败')
      console.error('Error updating context:', error)
    }
  } else {
    try {
      const config: ContextConfig = {
        name: contextForm.value.name,
        type: contextForm.value.type,
        host: buildDockerHost(contextForm.value),
        current: contextForm.value.current
      }
      await dockerApi.createContext(config)
      ElMessage.success('连接创建成功')
      dialogVisible.value = false
      await loadContexts()
    } catch (error) {
      ElMessage.error('创建连接失败')
      console.error('Error creating context:', error)
    }
  }
}

// 表单校验规则
const rules = {
  name: [{ required: true, message: '请输入连接名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择连接方式', trigger: 'change' }],
  host: [{ required: true, message: '请输入连接地址', trigger: 'blur' }]
}

onMounted(() => {
  loadContexts()
})

// 监听全局刷新事件
window.addEventListener('refresh-contexts', () => {
  loadContexts()
})
</script>

<style scoped>
.context-button {
  max-width: 200px;
  height: 32px;
  padding: 0 12px;
}

.context-button :deep(span) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 160px;
}

.context-button :deep(.el-button__text) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.el-dropdown-link {
  cursor: pointer;
}

.context-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.context-label {
  display: flex;
  align-items: center;
  gap: 4px;
}

.context-actions {
  display: flex;
  gap: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}

.context-item:hover .context-actions {
  opacity: 1;
}

.action-icon {
  padding: 2px;
  font-size: 14px;
  color: var(--el-color-primary);
  cursor: pointer;
  border-radius: 4px;
  
  &:hover {
    color: var(--el-color-primary-light-3);
    background: var(--el-color-primary-light-9);
  }
}

.is-active {
  color: var(--el-color-primary);
  font-weight: bold;
}

:deep(.el-dropdown-menu__item) {
  padding: 8px 12px;
  line-height: 1.5;
}

.el-empty {
  padding: 12px;
  margin: 0;
}

.el-dropdown-menu {
  min-width: 180px;
}

.empty-item {
  cursor: default !important;
}

.empty-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
}

.empty-icon {
  font-size: 16px;
}

:deep(.el-dropdown-menu__item.empty-item:hover) {
  background-color: transparent;
}

:deep(.el-dropdown-menu__item.is-disabled) {
  background-color: transparent;
}

:deep(.el-tag--small) {
  height: 20px;
  padding: 0 6px;
  font-size: 12px;
}

.action-wrapper {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.add-item {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
}

:deep(.el-dropdown-menu__item) {
  padding: 5px 12px;
}

.context-label {
  flex: 1;
  min-width: 0;
  padding: 5px 0;
  cursor: pointer;
}

:deep(.el-tag--small) {
  margin-left: 4px;
}
</style> 