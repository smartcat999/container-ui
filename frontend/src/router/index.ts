import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'
import ContextList from '@/views/ContextList.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: MainLayout,
      children: [
        {
          path: '',  // 默认路由
          name: 'contexts',
          component: ContextList
        },
        {
          path: 'containers',
          name: 'containers',
          component: () => import('@/views/ContainerList.vue')
        },
        {
          path: 'images',
          name: 'images',
          component: () => import('@/views/ImageList.vue')
        },
        {
          path: 'networks',
          name: 'networks',
          component: () => import('@/views/NetworkList.vue')
        },
        {
          path: 'volumes',
          name: 'volumes',
          component: () => import('@/views/VolumeList.vue')
        }
      ]
    }
  ]
})

export default router