import axios from 'axios'

const api = axios.create({
  baseURL: 'http://localhost:8080/api', // 后端API地址
  timeout: 5000
})

export interface Port {
  ip: string;
  privatePort: number;
  publicPort: number;
  type: string;
}

export interface Container {
  id: string;
  name: string;
  image: string;
  status: string;
  state: string;
  created: number;
  ports: Port[];
}

export interface Image {
  id: string;
  repository: string;
  tag: string;
  size: number;
  created: number;
}

export interface CreateContainerOptions {
  imageId: string;
  name?: string;
  command?: string;
  args?: string[];
  ports?: Array<{
    host: number;
    container: number;
  }>;
  env?: Array<{
    key: string;
    value: string;
  }>;
  volumes?: Array<{
    host: string;
    container: string;
    mode: string;
  }>;
  restartPolicy?: 'no' | 'on-failure' | 'always' | 'unless-stopped';
  networkMode?: 'bridge' | 'host' | 'none';
}

export interface Network {
  id: string;
  name: string;
  driver: string;
  scope: string;
  ipam: {
    config: Array<{
      subnet: string;
      gateway: string;
    }>;
  };
  created: string;
}

export interface Volume {
  name: string;
  driver: string;
  mountpoint: string;
  created: string;
  labels: Record<string, string>;
  scope: string;
  options: Record<string, string>;
}

export interface ContextConfig {
  name?: string;
  host: string;
  sshKeyFile?: string;
}

export interface DefaultContextConfig {
  host: string;
}

export const dockerApi = {
  // 获取容器列表
  getContainers() {
    return api.get<Container[]>('/containers')
  },
  // 获取镜像列表
  getImages() {
    return api.get<Image[]>('/images')
  },
  // 启动容器
  startContainer(id: string) {
    return api.post(`/containers/${id}/start`)
  },
  // 停止容器
  stopContainer(id: string) {
    return api.post(`/containers/${id}/stop`)
  },
  // 删除镜像
  deleteImage(id: string) {
    return api.delete(`/images/${id}`)
  },
  // 创建容器
  createContainer(options: CreateContainerOptions) {
    return api.post('/containers', options)
  },
  // 获取网络列表
  getNetworks() {
    return api.get<Network[]>('/networks')
  },
  // 获取网络详情
  getNetworkDetail(id: string) {
    return api.get(`/networks/${id}`)
  },
  // 删除网络
  deleteNetwork(id: string) {
    return api.delete(`/networks/${id}`)
  },
  // 获取数据卷列表
  getVolumes() {
    return api.get<Volume[]>('/volumes')
  },
  // 创建数据卷
  createVolume(data: { name: string; driver: string }) {
    return api.post('/volumes', data)
  },
  // 删除数据卷
  deleteVolume(name: string) {
    return api.delete(`/volumes/${name}`)
  },
  // 获取容器详情
  getContainerDetail(id: string) {
    return api.get(`/containers/${id}/json`)
  },
  // 获取镜像详情
  getImageDetail(id: string) {
    return api.get(`/images/${id}/json`)
  },
  getVolumeDetail(name: string) {
    return api.get(`/volumes/${name}`)
  },
  getContainerLogs(id: string) {
    return api.get(`/containers/${id}/logs`)
  },
  getContexts() {
    return api.get<string[]>('/contexts')
  },
  getCurrentContext() {
    return api.get<string>('/contexts/current')
  },
  switchContext(name: string) {
    return api.post(`/contexts/${name}/use`)
  },
  createContext(config: ContextConfig) {
    return api.post('/contexts', config)
  },
  // 获取默认 context 配置
  getDefaultContextConfig() {
    return api.get<{ host: string }>('/contexts/default/config')
  },
  // 修改更新默认 context 的方法
  updateDefaultContext(config: DefaultContextConfig) {
    return api.post('/contexts/default/config', config)  // 改用 POST 方法
  },
  getContextConfig(name: string) {
    return api.get<ContextConfig>(`/contexts/${name}/config`)
  },
  updateContextConfig(name: string, config: ContextConfig) {
    return api.post(`/contexts/${name}/config`, config)
  },
  deleteContext(name: string) {
    return api.delete(`/contexts/${name}`)
  },
  deleteContainer(id: string, force: boolean = false) {
    return api.delete(`/containers/${id}`, { params: { force } })
  }
}