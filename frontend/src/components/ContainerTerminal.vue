<template>
  <el-dialog
    v-model="visible"
    :title="'容器终端 - ' + containerId"
    width="80%"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :before-close="handleClose"
    class="terminal-dialog"
  >
    <div class="terminal-container" ref="terminalRef"></div>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import { createWebSocket } from '@/api/websocket'
import 'xterm/css/xterm.css'

const visible = ref(false)
const containerId = ref('')
const terminalRef = ref<HTMLElement>()
let terminal: Terminal | null = null
let socket: WebSocket | null = null
let fitAddon: FitAddon | null = null

const initTerminal = () => {
  if (!terminalRef.value) return

  // 初始化终端
  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#ffffff'
    },
    convertEol: true,
    cursorStyle: 'block',
    scrollback: 1000,
  })

  // 添加插件
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  // 打开终端
  terminal.open(terminalRef.value)
  fitAddon.fit()

  // 连接WebSocket
  socket = createWebSocket(`/containers/${containerId.value}/exec`)

  // 发送初始终端大小
  socket.onopen = () => {
    terminal?.writeln('Connected to container terminal...')
    if (terminal && socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({
        type: 'resize',
        cols: terminal.cols,
        rows: terminal.rows
      }))
    }
  }

  socket.onmessage = (event) => {
    try {
      if (event.data instanceof Blob) {
        // 处理二进制数据
        const reader = new FileReader()
        reader.onload = () => {
          if (typeof reader.result === 'string') {
            terminal?.write(reader.result)
          }
        }
        reader.readAsText(event.data)
      } else {
        // 处理文本数据
        terminal?.write(event.data)
      }
    } catch (error) {
      console.error('Error writing to terminal:', error)
    }
  }

  socket.onclose = () => {
    terminal?.writeln('\r\nConnection closed.')
  }

  socket.onerror = (error) => {
    console.error('WebSocket error:', error)
    terminal?.writeln('\r\nConnection error occurred.')
  }

  // 监听终端输入
  terminal.onData((data) => {
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({
        type: 'input',
        data: data
      }))
    }
  })

  // 监听终端大小变化
  const handleResize = () => {
    if (fitAddon && terminal) {
      fitAddon.fit()
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({
          type: 'resize',
          cols: terminal.cols,
          rows: terminal.rows
        }))
      }
    }
  }

  window.addEventListener('resize', handleResize)
  return () => window.removeEventListener('resize', handleResize)
}

const handleClose = () => {
  socket?.close()
  terminal?.dispose()
  terminal = null
  socket = null
  visible.value = false
}

const show = (id: string) => {
  containerId.value = id
  visible.value = true
  nextTick(() => {
    initTerminal()
  })
}

onBeforeUnmount(() => {
  handleClose()
})

defineExpose({
  show
})
</script>

<style scoped>
.terminal-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.terminal-container {
  min-height: 400px;
  height: 60vh;
  background-color: #1e1e1e;
  padding: 8px;
  overflow: hidden;
}

:deep(.xterm) {
  padding: 8px;
  height: 100%;
}

:deep(.xterm-viewport) {
  overflow-y: auto !important;
}
</style> 