// API 基础配置
export const API_CONFIG = {
  BASE_URL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  
  // 获取 HTTP API 基础地址
  getHttpBaseUrl() {
    return this.BASE_URL
  },
  
  // 获取 WebSocket 基础地址
  getWsBaseUrl() {
    return this.BASE_URL.replace(/^http/, 'ws')
  }
} 