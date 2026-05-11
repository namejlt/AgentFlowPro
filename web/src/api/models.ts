import http from './http'
import type { ApiResponse, LlmModelItem, PaginationParams } from '@/types'

export function getModels(params?: PaginationParams & { enabled?: boolean }) {
  return http.get<ApiResponse<LlmModelItem[]>>('/api/v1/models', { params })
}

export function getModel(id: string) {
  return http.get<ApiResponse<LlmModelItem>>(`/api/v1/models/${id}`)
}

export function createModel(data: Partial<LlmModelItem> & { api_key?: string }) {
  return http.post<ApiResponse<LlmModelItem>>('/api/v1/models', data)
}

export function updateModel(id: string, data: Partial<LlmModelItem> & { api_key?: string }) {
  return http.put<ApiResponse<LlmModelItem>>(`/api/v1/models/${id}`, data)
}

export function deleteModel(id: string) {
  return http.delete<ApiResponse<void>>(`/api/v1/models/${id}`)
}

export function testModel(id: string) {
  return http.post<ApiResponse<{ success: boolean; latency_ms: number; error?: string }>>(`/api/v1/models/${id}/test`)
}

export function setDefaultModel(id: string) {
  return http.patch<ApiResponse<void>>(`/api/v1/models/${id}/default`)
}
