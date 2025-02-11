<template>
  <el-dialog
    v-model="dialogVisible"
    title="创建容器"
    width="600px"
  >
    <el-form :model="form" label-width="120px">
      <el-form-item label="容器名称">
        <el-input v-model="form.name" placeholder="请输入容器名称" />
      </el-form-item>

      <el-form-item label="启动命令">
        <el-input 
          v-model="form.command" 
          placeholder="例如: /bin/bash"
          type="text"
        />
      </el-form-item>

      <el-form-item label="启动参数">
        <div v-for="(arg, index) in form.args" :key="index" class="args-list">
          <el-input v-model="arg.value" placeholder="参数" />
          <el-button type="danger" @click="removeArg(index)">
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
        <el-button type="primary" @click="addArg">
          <el-icon><Plus /></el-icon>
          添加参数
        </el-button>
      </el-form-item>

      <el-form-item label="端口映射">
        <div v-for="(port, index) in form.ports" :key="index" class="port-mapping">
          <el-input-number 
            v-model="port.host" 
            :min="1" 
            :max="65535" 
            placeholder="主机端口"
          />
          <span class="port-separator">:</span>
          <el-input-number 
            v-model="port.container" 
            :min="1" 
            :max="65535" 
            placeholder="容器端口"
          />
          <el-button type="danger" @click="removePort(index)">
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
        <el-button type="primary" @click="addPort">
          <el-icon><Plus /></el-icon>
          添加端口映射
        </el-button>
      </el-form-item>

      <el-form-item label="环境变量">
        <div v-for="(env, index) in form.env" :key="index" class="env-variable">
          <el-input v-model="env.key" placeholder="键" />
          <span class="env-separator">=</span>
          <el-input v-model="env.value" placeholder="值" />
          <el-button type="danger" @click="removeEnv(index)">
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
        <el-button type="primary" @click="addEnv">
          <el-icon><Plus /></el-icon>
          添加环境变量
        </el-button>
      </el-form-item>

      <el-form-item label="数据卷">
        <div v-for="(volume, index) in form.volumes" :key="index" class="volume-mapping">
          <el-input v-model="volume.host" placeholder="主机路径" />
          <span class="volume-separator">:</span>
          <el-input v-model="volume.container" placeholder="容器路径" />
          <el-select v-model="volume.mode" placeholder="权限">
            <el-option label="读写" value="rw" />
            <el-option label="只读" value="ro" />
          </el-select>
          <el-button type="danger" @click="removeVolume(index)">
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
        <el-button type="primary" @click="addVolume">
          <el-icon><Plus /></el-icon>
          添加数据卷
        </el-button>
      </el-form-item>

      <el-form-item label="重启策略">
        <el-select v-model="form.restartPolicy">
          <el-option label="不自动重启" value="no" />
          <el-option label="失败时重启" value="on-failure" />
          <el-option label="除非手动停止，总是重启" value="always" />
          <el-option label="无论如何都重启" value="unless-stopped" />
        </el-select>
      </el-form-item>

      <el-form-item label="网络模式">
        <el-select v-model="form.networkMode">
          <el-option label="桥接" value="bridge" />
          <el-option label="主机" value="host" />
          <el-option label="无网络" value="none" />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="loading">
          创建
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, defineExpose } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'
import { dockerApi } from '@/api/docker'
import { useContextStore } from '@/store/context'

interface PortMapping {
  host: number
  container: number
}

interface ContainerForm {
  imageId: string
  name: string
  command: string
  args: Array<{ value: string }>
  ports: PortMapping[]
  env: Array<{ key: string; value: string }>
  volumes: Array<{ host: string; container: string; mode: string }>
  restartPolicy: 'no' | 'on-failure' | 'always' | 'unless-stopped'
  networkMode: 'bridge' | 'host' | 'none'
}

const dialogVisible = ref(false)
const loading = ref(false)
const form = ref<ContainerForm>({
  imageId: '',
  name: '',
  command: '',
  args: [],
  ports: [],
  env: [],
  volumes: [],
  restartPolicy: 'no',
  networkMode: 'bridge'
})

const contextStore = useContextStore()

const addPort = () => {
  form.value.ports.push({ host: 0, container: 0 })
}

const removePort = (index: number) => {
  form.value.ports.splice(index, 1)
}

const addEnv = () => {
  form.value.env.push({ key: '', value: '' })
}

const removeEnv = (index: number) => {
  form.value.env.splice(index, 1)
}

const addVolume = () => {
  form.value.volumes.push({ host: '', container: '', mode: 'rw' })
}

const removeVolume = (index: number) => {
  form.value.volumes.splice(index, 1)
}

const addArg = () => {
  form.value.args.push({ value: '' })
}

const removeArg = (index: number) => {
  form.value.args.splice(index, 1)
}

const resetForm = () => {
  form.value = {
    imageId: '',
    name: '',
    command: '',
    args: [],
    ports: [],
    env: [],
    volumes: [],
    restartPolicy: 'no',
    networkMode: 'bridge'
  }
}

const handleCreate = async () => {
  if (!form.value.imageId) {
    ElMessage.warning('镜像ID不能为空')
    return
  }

  const invalidArgs = form.value.args.some(arg => !arg.value.trim())
  if (invalidArgs) {
    ElMessage.warning('启动参数不能为空')
    return
  }

  const invalidPorts = form.value.ports.some(p => p.host <= 0 || p.container <= 0)
  if (invalidPorts) {
    ElMessage.warning('请填写有效的端口映射')
    return
  }

  const invalidEnv = form.value.env.some(e => !e.key)
  if (invalidEnv) {
    ElMessage.warning('环境变量的键不能为空')
    return
  }

  const invalidVolumes = form.value.volumes.some(v => !v.host || !v.container)
  if (invalidVolumes) {
    ElMessage.warning('请填写完整的数据卷映射信息')
    return
  }

  loading.value = true
  try {
    await dockerApi.createContainer(contextStore.getCurrentContext(), {
      imageId: form.value.imageId,
      name: form.value.name,
      command: form.value.command,
      args: form.value.args.map(arg => arg.value),
      ports: form.value.ports,
      env: form.value.env,
      volumes: form.value.volumes,
      restartPolicy: form.value.restartPolicy,
      networkMode: form.value.networkMode
    })
    ElMessage.success('容器创建成功')
    dialogVisible.value = false
    resetForm()
  } catch (error) {
    ElMessage.error('容器创建失败')
    console.error('Error creating container:', error)
  } finally {
    loading.value = false
  }
}

const show = (imageId: string) => {
  form.value.imageId = imageId
  dialogVisible.value = true
}

defineExpose({
  show
})
</script>

<style scoped>
.port-mapping,
.env-variable,
.volume-mapping {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  align-items: center;
}

.port-separator,
.env-separator,
.volume-separator {
  margin: 0 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

:deep(.el-input-number) {
  width: 120px;
}

:deep(.el-select) {
  width: 120px;
}

.args-list {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  align-items: center;
}
</style> 