import http from './http'
import type { ApiResponse, DataSourceItem, PaginationParams, ParamSchema } from '@/types'

export function getDataSources(params?: PaginationParams & { type?: string; category?: string }) {
  return http.get<ApiResponse<DataSourceItem[]>>('/api/v1/datasources', { params })
}

export function getDataSource(id: string) {
  return http.get<ApiResponse<DataSourceItem>>(`/api/v1/datasources/${id}`)
}

export function createDataSource(data: Partial<DataSourceItem>) {
  return http.post<ApiResponse<DataSourceItem>>('/api/v1/datasources', data)
}

export function updateDataSource(id: string, data: Partial<DataSourceItem>) {
  return http.put<ApiResponse<DataSourceItem>>(`/api/v1/datasources/${id}`, data)
}

export function cloneDataSource(id: string) {
  return http.post<ApiResponse<DataSourceItem>>(`/api/v1/datasources/${id}/clone`)
}

export function deleteDataSource(id: string) {
  return http.delete<ApiResponse<void>>(`/api/v1/datasources/${id}`)
}

export function patchDataSourceStatus(id: string, enabled: boolean) {
  return http.patch<ApiResponse<void>>(`/api/v1/datasources/${id}/status`, { enabled })
}

export function testDataSource(id: string, params?: ParamSchema[]) {
  return http.post<ApiResponse<{ ok: boolean; extracted?: string; raw?: string; from_cache?: boolean; error?: string }>>(`/api/v1/datasources/${id}/test`, { params })
}

export function uploadFile(formData: FormData) {
  return http.post<ApiResponse<{ file_id: string; url: string }>>('/api/v1/files', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}
