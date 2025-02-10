import axios from 'axios'

const getBaseUrl = () => {
  const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'
  return baseURL.replace(/^http/, 'ws')
}

export const createWebSocket = (path: string): WebSocket => {
  const baseURL = getBaseUrl()
  const wsUrl = `${baseURL}${path}`
  return new WebSocket(wsUrl)
} 