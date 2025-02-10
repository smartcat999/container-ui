import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'
import ContainerList from '@/views/ContainerList.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: MainLayout,
      children: [
        {
          path: '',
          name: 'containers',
          component: ContainerList
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