import { API_CONFIG } from '@/config'

export const createWebSocket = (path: string): WebSocket => {
  const baseURL = API_CONFIG.getWsBaseUrl()
  const wsUrl = `${baseURL}${path}`
  return new WebSocket(wsUrl)
} 