import { API_CONFIG } from '@/config'

export const createWebSocket = (contextName: string, path: string): WebSocket => {
  const baseURL = API_CONFIG.getWsBaseUrl()
  const wsUrl = `${baseURL}/contexts/${contextName}${path}`
  return new WebSocket(wsUrl)
} 