<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '200px'" class="aside">
      <div class="logo">
        <h2 v-if="!isCollapse">Docker Manager</h2>
        <h2 v-else>D</h2>
      </div>
      <el-menu
        :router="true"
        :default-active="route.path"
        class="menu"
        :collapse="isCollapse"
        background-color="#304156"
        text-color="#fff"
        active-text-color="#409EFF"
      >
        <el-menu-item index="/">
          <el-icon><Connection /></el-icon>
          <span>连接管理</span>
        </el-menu-item>
        <el-menu-item index="/containers">
          <el-icon><Box /></el-icon>
          <span>容器管理</span>
        </el-menu-item>
        <el-menu-item index="/images">
          <el-icon><Picture /></el-icon>
          <span>镜像管理</span>
        </el-menu-item>
        <el-menu-item index="/networks">
          <el-icon><Share /></el-icon>
          <span>网络管理</span>
        </el-menu-item>
        <el-menu-item index="/volumes">
          <el-icon><Files /></el-icon>
          <span>数据卷管理</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <!-- 顶部信息栏 -->
      <el-header class="header">
        <div class="header-left">
          <el-icon class="toggle-sidebar" @click="toggleSidebar">
            <Fold v-if="!isCollapse" />
            <Expand v-else />
          </el-icon>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item>首页</el-breadcrumb-item>
            <el-breadcrumb-item>{{ currentMenuTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <context-manager />
          <el-divider direction="vertical" />
          <el-dropdown>
            <span class="user-info">
              <el-avatar :size="32" />
              管理员
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>个人信息</el-dropdown-item>
                <el-dropdown-item>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 主体内容 -->
      <el-main class="main">
        <router-view />
      </el-main>

      <!-- 底部信息栏 -->
      <el-footer class="footer">
        <p>Docker Web Manager ©2024 Created by Your Name</p>
      </el-footer>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  Connection,
  Box,
  Picture,
  Share,
  Files,
  Fold,
  Expand
} from '@element-plus/icons-vue'
import ContextManager from '@/components/ContextManager.vue'

const route = useRoute()
const isCollapse = ref(false)

// 计算当前菜单标题
const currentMenuTitle = computed(() => {
  switch (route.path) {
    case '/':
      return '容器管理'
    case '/images':
      return '镜像管理'
    case '/networks':
      return '网络管理'
    case '/volumes':
      return '数据卷管理'
    default:
      return '容器管理'
  }
})

const toggleSidebar = () => {
  isCollapse.value = !isCollapse.value
}

// 监听 context 变更事件，刷新当前页面数据
window.addEventListener('context-changed', () => {
  // 触发当前页面的数据刷新
  window.dispatchEvent(new CustomEvent('refresh-data'))
})
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.aside {
  background-color: #304156;
  color: #fff;
  transition: width 0.3s;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  border-bottom: 1px solid #1f2d3d;
  overflow: hidden;
  transition: all 0.3s;
}

.menu {
  border: none;
}

.menu:not(.el-menu--collapse) {
  width: 200px;
}

.header {
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.toggle-sidebar {
  font-size: 20px;
  cursor: pointer;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.main {
  background-color: #f0f2f5;
  padding: 20px;
}

.footer {
  text-align: center;
  background-color: #fff;
  color: #666;
  border-top: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style> 