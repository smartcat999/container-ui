import axios from 'axios'
import { API_CONFIG } from '@/config'

const request = axios.create({
  baseURL: API_CONFIG.getHttpBaseUrl(),
  timeout: API_CONFIG.getHttpTimeout()
})

export default request

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
  created: string;
  ports: string[];
  loading?: boolean;
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

export interface ContainerConfig {
  name: string
  image: string
  ports: Array<{
    host: number
    container: number
  }>
  env: Array<{
    key: string
    value: string
  }>
  volumes: Array<{
    host: string
    container: string
    mode: string
  }>
  restart: 'no' | 'on-failure' | 'always' | 'unless-stopped'
  networkMode: 'bridge' | 'host' | 'none'
  // ... 其他配置项
}

export const dockerApi = {
  // 获取容器列表
  getContainers() {
    return request.get<Container[]>('/containers')
  },
  // 获取镜像列表
  getImages() {
    return request.get<Image[]>('/images')
  },
  // 启动容器
  startContainer(id: string) {
    return request.post(`/containers/${id}/start`)
  },
  // 停止容器
  stopContainer(id: string) {
    return request.post(`/containers/${id}/stop`)
  },
  // 删除镜像
  deleteImage(id: string) {
    return request.delete(`/images/${id}`)
  },
  // 创建容器
  createContainer(options: CreateContainerOptions) {
    return request.post('/containers', options)
  },
  // 获取网络列表
  getNetworks() {
    return request.get<Network[]>('/networks')
  },
  // 获取网络详情
  getNetworkDetail(id: string) {
    return request.get(`/networks/${id}`)
  },
  // 删除网络
  deleteNetwork(id: string) {
    return request.delete(`/networks/${id}`)
  },
  // 获取数据卷列表
  getVolumes() {
    return request.get<Volume[]>('/volumes')
  },
  // 创建数据卷
  createVolume(data: { name: string; driver: string }) {
    return request.post('/volumes', data)
  },
  // 删除数据卷
  deleteVolume(name: string) {
    return request.delete(`/volumes/${name}`)
  },
  // 获取容器详情
  getContainerDetail(id: string) {
    return request.get(`/containers/${id}/json`)
  },
  // 获取镜像详情
  getImageDetail(id: string) {
    return request.get(`/images/${id}/json`)
  },
  getVolumeDetail(name: string) {
    return request.get(`/volumes/${name}`)
  },
  getContainerLogs(id: string) {
    return request.get(`/containers/${id}/logs`)
  },
  getContexts() {
    return request.get<string[]>('/contexts')
  },
  getCurrentContext() {
    return request.get<string>('/contexts/current')
  },
  switchContext(name: string) {
    return request.post(`/contexts/${name}/use`)
  },
  createContext(config: ContextConfig) {
    return request.post('/contexts', config)
  },
  // 获取默认 context 配置
  getDefaultContextConfig() {
    return request.get<{ host: string }>('/contexts/default/config')
  },
  // 修改更新默认 context 的方法
  updateDefaultContext(config: DefaultContextConfig) {
    return request.post('/contexts/default/config', config)  // 改用 POST 方法
  },
  getContextConfig(name: string) {
    return request.get<ContextConfig>(`/contexts/${name}/config`)
  },
  updateContextConfig(name: string, config: ContextConfig) {
    return request.post(`/contexts/${name}/config`, config)
  },
  deleteContext(name: string) {
    return request.delete(`/contexts/${name}`)
  },
  deleteContainer(id: string, force: boolean = false) {
    return request.delete(`/containers/${id}`, { params: { force } })
  }
}