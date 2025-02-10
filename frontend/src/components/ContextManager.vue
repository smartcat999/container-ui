<template>
  <el-dropdown trigger="click" @command="handleContextCommand">
    <el-button type="primary">
      <span>{{ currentContext }}</span>
      <el-icon class="el-icon--right">
        <arrow-down />
      </el-icon>
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="context in contexts"
          :key="context"
          :class="{ 'is-active': context === currentContext }"
        >
          <div class="context-item">
            <span @click="handleContextCommand({ type: 'switch', name: context })">
              {{ context }}
            </span>
            <div class="context-actions">
              <el-icon 
                class="action-icon"
                @click.stop="showEditDialog(context)"
              >
                <Edit />
              </el-icon>
              <el-icon 
                v-if="context !== 'default'"
                class="action-icon"
                @click.stop="confirmDelete(context)"
              >
                <Delete />
              </el-icon>
            </div>
          </div>
        </el-dropdown-item>
        <el-dropdown-item divided @click.stop="showAddContextDialog">
          <el-icon><Plus /></el-icon>
          添加 Context
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>

  <!-- 添加/编辑 Context 对话框 -->
  <el-dialog
    v-model="dialogVisible"
    :title="isEditing ? '编辑 Docker Context' : '添加 Docker Context'"
    width="500px"
  >
    <el-form :model="contextForm" label-width="120px">
      <el-form-item label="Context 名称" v-if="!isEditing">
        <el-input v-model="contextForm.name" placeholder="请输入名称" />
      </el-form-item>
      
      <el-form-item label="连接方式">
        <el-radio-group v-model="contextForm.type">
          <el-radio label="tcp">TCP</el-radio>
          <el-radio label="socket" :disabled="contextForm.name !== 'default'">Socket</el-radio>
        </el-radio-group>
      </el-form-item>

      <template v-if="contextForm.type === 'tcp'">
        <el-form-item label="主机地址">
          <el-input v-model="contextForm.host" placeholder="例如: 192.168.1.100" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="contextForm.port" :min="1" :max="65535" :step="1" :default-value="2375" />
        </el-form-item>
      </template>

      <template v-if="contextForm.type === 'socket'">
        <el-form-item label="Socket 路径">
          <el-input 
            v-model="contextForm.socketPath" 
            placeholder="例如: /var/run/docker.sock"
          >
            <template #append>
              <el-tooltip content="使用默认 socket 路径" placement="top">
                <el-button @click="useDefaultSocket">默认</el-button>
              </el-tooltip>
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
import { ArrowDown, Plus, Edit, Delete } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'

interface ContextCommand {
  type: 'switch';
  name: string;
}

const currentContext = ref('default')
const contexts = ref<string[]>([])
const dialogVisible = ref(false)
const isEditing = ref(false)
const contextForm = ref({
  name: '',
  type: 'tcp',
  host: '',
  port: 2375,
  socketPath: '/var/run/docker.sock'
})

const loadContexts = async () => {
  try {
    const response = await dockerApi.getContexts()
    contexts.value = response.data
    const current = await dockerApi.getCurrentContext()
    currentContext.value = current.data
  } catch (error) {
    ElMessage.error('加载 Docker Context 失败')
    console.error('Error loading contexts:', error)
  }
}

const handleContextCommand = async (command: ContextCommand) => {
  if (command.type === 'switch') {
    try {
      await dockerApi.switchContext(command.name)
      currentContext.value = command.name
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
    socketPath: '/var/run/docker.sock'
  }
  dialogVisible.value = true
}

const showEditDialog = async (contextName: string) => {
  isEditing.value = true
  try {
    const response = await dockerApi.getContextConfig(contextName)
    contextForm.value.name = contextName
    parseDockerHost(response.data.host)
    dialogVisible.value = true
  } catch (error) {
    ElMessage.error('获取 Context 配置失败')
    console.error('Error getting context config:', error)
  }
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
        if (currentContext.value === contextName) {
          await handleContextCommand({ type: 'switch', name: 'default' })
        }
        loadContexts()
      } catch (error) {
        ElMessage.error('删除 Context 失败')
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
    return `tcp://${form.host}:${form.port}`
  } else {
    return `unix://${form.socketPath}`
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
  // 验证必填字段
  if (!contextForm.value.name && !isEditing.value) {
    ElMessage.warning('请填写 Context 名称')
    return
  }

  if (contextForm.value.type === 'tcp') {
    if (!contextForm.value.host || !contextForm.value.port) {
      ElMessage.warning('请填写完整的 TCP 连接信息')
      return
    }
  }

  const config = {
    name: contextForm.value.name,
    host: buildDockerHost(contextForm.value)
  }

  try {
    if (isEditing.value) {
      await dockerApi.updateContextConfig(contextForm.value.name, { host: config.host })
      ElMessage.success('更新 Context 成功')
    } else {
      await dockerApi.createContext(config)
      ElMessage.success('添加 Context 成功')
      
      ElMessageBox.confirm(
        '是否切换到新创建的 Context？',
        '切换 Context',
        {
          confirmButtonText: '确认',
          cancelButtonText: '取消',
          type: 'info',
        }
      )
        .then(async () => {
          await handleContextCommand({ type: 'switch', name: contextForm.value.name })
        })
        .catch(() => {
          // 用户取消切换，不做任何操作
        })
    }
    
    dialogVisible.value = false
    loadContexts()
  } catch (error) {
    ElMessage.error(isEditing.value ? '更新 Context 失败' : '添加 Context 失败')
    console.error('Error saving context:', error)
  }
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
.context-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  cursor: default;
}

.context-actions {
  display: flex;
  gap: 8px;
}

.action-icon {
  cursor: pointer;
  font-size: 14px;
  color: var(--el-color-primary);
  
  &:hover {
    color: var(--el-color-primary-light-3);
  }
}

.is-active {
  color: var(--el-color-primary);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

:deep(.el-dropdown-menu__item) {
  padding: 5px 12px;
}

:deep(.el-input-group__append) {
  padding: 0;
  .el-button {
    margin: 0;
    border: none;
  }
}
</style> 