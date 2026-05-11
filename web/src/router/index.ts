import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/auth/LoginView.vue'),
    meta: { public: true },
  },
  {
    path: '/',
    component: () => import('@/components/layout/AppShell.vue'),
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/dashboard/DashboardView.vue') },
      { path: 'workflows', name: 'WorkflowList', component: () => import('@/views/workflows/WorkflowListView.vue') },
      { path: 'workflows/:id/edit', name: 'WorkflowEditor', component: () => import('@/views/workflows/WorkflowEditorView.vue') },
      { path: 'tasks', name: 'TaskList', component: () => import('@/views/tasks/TaskListView.vue') },
      { path: 'tasks/:id', name: 'TaskMonitor', component: () => import('@/views/tasks/TaskMonitorView.vue') },
      { path: 'agents', name: 'AgentList', component: () => import('@/views/agents/AgentListView.vue') },
      { path: 'agents/:id', name: 'AgentEditor', component: () => import('@/views/agents/AgentEditorView.vue') },
      { path: 'datasources', name: 'DataSourceList', component: () => import('@/views/datasources/DataSourceListView.vue') },
      { path: 'datasources/:id', name: 'DataSourceEditor', component: () => import('@/views/datasources/DataSourceEditorView.vue') },
      { path: 'models', name: 'ModelList', component: () => import('@/views/models/ModelListView.vue') },
      { path: 'reports', name: 'ReportList', component: () => import('@/views/reports/ReportListView.vue') },
      { path: 'reports/:id', name: 'ReportDetail', component: () => import('@/views/reports/ReportDetailView.vue') },
      { path: 'settings', name: 'SystemConfig', component: () => import('@/views/settings/SystemConfigView.vue') },
      { path: 'settings/users', name: 'UserAdmin', component: () => import('@/views/settings/UserAdminView.vue') },
      { path: 'settings/audit', name: 'AuditLog', component: () => import('@/views/settings/AuditLogView.vue') },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
