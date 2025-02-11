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

// 定义连接类型
export type ContextType = 'tcp' | 'socket'

// 更新 ContextConfig 接口
export interface ContextConfig {
  name: string
  type: ContextType
  host: string
  current: boolean
}

// 定义表单类型
export interface ContextForm {
  name: string
  type: ContextType
  host: string
  port: number
  socketPath: string
  current: boolean
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
  // Context 相关 API - 不需要 context 参数
  getContexts() {
    return request.get<ContextConfig[]>('/contexts')
  },
  createContext(data: ContextConfig) {
    return request.post('/contexts', data)
  },
  updateContextConfig(name: string, data: ContextConfig) {
    return request.put(`/contexts/${name}`, data)
  },
  deleteContext(name: string) {
    return request.delete(`/contexts/${name}`)
  },
  getContextConfig(name: string) {
    return request.get<ContextConfig>(`/contexts/${name}`)
  },

  // 需要 context 参数的资源 API
  // 容器相关
  getContainers(contextName: string) {
    return request.get<Container[]>(`/contexts/${contextName}/containers`)
  },
  startContainer(contextName: string, id: string) {
    return request.post(`/contexts/${contextName}/containers/${id}/start`)
  },
  stopContainer(contextName: string, id: string) {
    return request.post(`/contexts/${contextName}/containers/${id}/stop`)
  },
  deleteContainer(contextName: string, id: string, force: boolean = false) {
    return request.delete(`/contexts/${contextName}/containers/${id}`, { params: { force } })
  },
  getContainerDetail(contextName: string, id: string) {
    return request.get(`/contexts/${contextName}/containers/${id}/json`)
  },
  getContainerLogs(contextName: string, id: string) {
    return request.get(`/contexts/${contextName}/containers/${id}/logs`)
  },

  // 镜像相关
  getImages(contextName: string) {
    return request.get<Image[]>(`/contexts/${contextName}/images`)
  },
  deleteImage(contextName: string, id: string) {
    return request.delete(`/contexts/${contextName}/images/${id}`)
  },
  createContainer(contextName: string, options: CreateContainerOptions) {
    return request.post(`/contexts/${contextName}/containers`, options)
  },
  getImageDetail(contextName: string, id: string) {
    return request.get(`/contexts/${contextName}/images/${id}/json`)
  },

  // 网络相关
  getNetworks(contextName: string) {
    return request.get<Network[]>(`/contexts/${contextName}/networks`)
  },
  getNetworkDetail(contextName: string, id: string) {
    return request.get(`/contexts/${contextName}/networks/${id}`)
  },
  deleteNetwork(contextName: string, id: string) {
    return request.delete(`/contexts/${contextName}/networks/${id}`)
  },

  // 数据卷相关
  getVolumes(contextName: string) {
    return request.get<Volume[]>(`/contexts/${contextName}/volumes`)
  },
  getVolumeDetail(contextName: string, name: string) {
    return request.get(`/contexts/${contextName}/volumes/${name}`)
  },
  deleteVolume(contextName: string, name: string) {
    return request.delete(`/contexts/${contextName}/volumes/${name}`)
  }
}