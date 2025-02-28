import { defineStore } from 'pinia'
import { ref } from 'vue'
import { dockerApi } from '@/api/docker'
import type { ContextConfig } from '@/api/docker'

export const useContextStore = defineStore('context', () => {
  const currentContext = ref<string>('')
  const contextList = ref<ContextConfig[]>([])

  function setCurrentContext(name: string) {
    currentContext.value = name
  }

  function getCurrentContext() {
    return currentContext.value
  }

  async function loadContexts() {
    try {
      const response = await dockerApi.getContexts()
      if (!response.data) {
        contextList.value = []
        return
      }
      
      contextList.value = response.data.map(ctx => ({
        ...ctx,
        current: ctx.name === currentContext.value
      }))
    } catch (error) {
      console.error('Error loading contexts:', error)
      contextList.value = []
    }
  }

  return {
    currentContext,
    contextList,
    setCurrentContext,
    getCurrentContext,
    loadContexts
  }
}) 