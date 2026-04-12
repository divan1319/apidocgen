import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '@/views/Dashboard.vue'
import ProjectForm from '@/views/ProjectForm.vue'
import DocsViewer from '@/views/DocsViewer.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: Dashboard },
    { path: '/projects/new', name: 'project-new', component: ProjectForm },
    { path: '/projects/:slug/edit', name: 'project-edit', component: ProjectForm },
    { path: '/viewer/:slug', name: 'docs-viewer', component: DocsViewer },
  ],
})

export default router
