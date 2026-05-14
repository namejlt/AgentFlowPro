import http from './http'
import type { ApiResponse, WorkflowItem, PaginationParams, ImportMatchReport, ImportWorkflowResult, ShareInfo } from '@/types'

export function getWorkflows(params?: PaginationParams & { visibility?: string; archived?: boolean }) {
  return http.get<ApiResponse<WorkflowItem[]>>('/api/v1/workflows', { params })
}

export function getWorkflow(id: string) {
  return http.get<ApiResponse<WorkflowItem>>(`/api/v1/workflows/${id}`)
}

export function createWorkflow(data: Partial<WorkflowItem>) {
  return http.post<ApiResponse<WorkflowItem>>('/api/v1/workflows', data)
}

export function updateWorkflow(id: string, data: Partial<WorkflowItem>) {
  return http.put<ApiResponse<WorkflowItem>>(`/api/v1/workflows/${id}`, data)
}

export function cloneWorkflow(id: string) {
  return http.post<ApiResponse<WorkflowItem>>(`/api/v1/workflows/${id}/clone`)
}

export function deleteWorkflow(id: string) {
  return http.delete<ApiResponse<void>>(`/api/v1/workflows/${id}`)
}

export function getWorkflowVersions(id: string) {
  return http.get<ApiResponse<{ version: number; created_at: string; created_by: string }[]>>(`/api/v1/workflows/${id}/versions`)
}

export function rollbackWorkflow(id: string, version: number) {
  return http.post<ApiResponse<WorkflowItem>>(`/api/v1/workflows/${id}/versions/${version}/rollback`)
}

export function exportWorkflow(id: string) {
  return http.get(`/api/v1/workflows/${id}/export`, { responseType: 'blob' })
}

export function importWorkflow(data: any) {
  return http.post<ApiResponse<ImportWorkflowResult>>('/api/v1/workflows/import', data)
}

export function confirmImport(data: { session_id: string; bindings: Record<string, string> }) {
  return http.post<ApiResponse<WorkflowItem>>('/api/v1/workflows/import/confirm', data)
}

export function shareWorkflow(id: string, data?: { expires_at?: string }) {
  return http.post<ApiResponse<ShareInfo>>(`/api/v1/workflows/${id}/share`, data)
}

export function cloneByCode(code: string) {
  return http.post<ApiResponse<WorkflowItem>>('/api/v1/workflows/clone-by-code', { code })
}

export function patchWorkflowVisibility(id: string, visibility: string) {
  return http.patch<ApiResponse<void>>(`/api/v1/workflows/${id}/visibility`, { visibility })
}
