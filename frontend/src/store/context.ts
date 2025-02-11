import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ContextConfig } from '@/api/docker'

export const useContextStore = defineStore('context', () => {
  const currentContext = ref<string>('')

  const setCurrentContext = (contextName: string) => {
    currentContext.value = contextName
    localStorage.setItem('currentContext', contextName)
  }

  const getCurrentContext = () => {
    if (!currentContext.value) {
      currentContext.value = localStorage.getItem('currentContext') || ''
    }
    return currentContext.value
  }

  return {
    currentContext,
    setCurrentContext,
    getCurrentContext
  }
}) 