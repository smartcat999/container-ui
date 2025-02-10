const getApiBaseUrl = (): string => {
    // 使用当前浏览器地址构建 API URL
    const location = window.location
    const baseUrl = `${location.protocol}//${location.host}/api`
    console.log('Using API_BASE_URL from browser:', baseUrl)
    return baseUrl
  }


// API 基础配置
export const API_CONFIG = {
  BASE_URL: getApiBaseUrl(),

  // 获取 HTTP API 基础地址
  getHttpBaseUrl() {
    return this.BASE_URL
  },

  getHttpTimeout() {
    return 10000;
  },

  // 获取 WebSocket 基础地址
  getWsBaseUrl() {
    return this.BASE_URL.replace(/^http/, 'ws')
  }
}
