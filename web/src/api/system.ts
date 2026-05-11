import http from './http'
import type { ApiResponse, DashboardStats, AuditLogItem, SystemConfigItem, PaginationParams } from '@/types'

export function getDashboard() {
  return http.get<ApiResponse<DashboardStats>>('/api/v1/system/dashboard')
}

export function getSystemConfig() {
  return http.get<ApiResponse<SystemConfigItem[]>>('/api/v1/system/config')
}

export function updateSystemConfig(data: Record<string, any>) {
  return http.patch<ApiResponse<void>>('/api/v1/system/config', data)
}

export function getAuditLogs(params?: PaginationParams & { action?: string; user_id?: string }) {
  return http.get<ApiResponse<AuditLogItem[]>>('/api/v1/system/audit-logs', { params })
}
