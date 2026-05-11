import http from './http'
import type { ApiResponse, AgentItem, PaginationParams } from '@/types'

export function getAgents(params?: PaginationParams & { enabled?: boolean }) {
  return http.get<ApiResponse<AgentItem[]>>('/api/v1/agents', { params })
}

export function getAgent(id: string) {
  return http.get<ApiResponse<AgentItem>>(`/api/v1/agents/${id}`)
}

export function createAgent(data: Partial<AgentItem>) {
  return http.post<ApiResponse<AgentItem>>('/api/v1/agents', data)
}

export function updateAgent(id: string, data: Partial<AgentItem>) {
  return http.put<ApiResponse<AgentItem>>(`/api/v1/agents/${id}`, data)
}

export function cloneAgent(id: string) {
  return http.post<ApiResponse<AgentItem>>(`/api/v1/agents/${id}/clone`)
}

export function deleteAgent(id: string) {
  return http.delete<ApiResponse<void>>(`/api/v1/agents/${id}`)
}

export function previewAgent(id: string, input?: Record<string, any>) {
  return http.post<ApiResponse<{ output: string; tokens_used: number }>>(`/api/v1/agents/${id}/preview`, { input })
}
