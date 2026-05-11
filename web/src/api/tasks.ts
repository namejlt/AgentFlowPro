import http from './http'
import type { ApiResponse, TaskItem, TaskStepItem, PaginationParams } from '@/types'

export function getTasks(params?: PaginationParams & { status?: string; workflow_id?: string }) {
  return http.get<ApiResponse<TaskItem[]>>('/api/v1/tasks', { params })
}

export function getTask(id: string) {
  return http.get<ApiResponse<TaskItem>>(`/api/v1/tasks/${id}`)
}

export function createTask(data: { workflow_id: string; input_params?: Record<string, any>; mode?: string }) {
  return http.post<ApiResponse<TaskItem>>('/api/v1/tasks', data)
}

export function stopTask(id: string) {
  return http.post<ApiResponse<void>>(`/api/v1/tasks/${id}/stop`)
}

export function rerunTask(id: string) {
  return http.post<ApiResponse<TaskItem>>(`/api/v1/tasks/${id}/rerun`)
}

export function getTaskSteps(id: string) {
  return http.get<ApiResponse<TaskStepItem[]>>(`/api/v1/tasks/${id}/steps`)
}

export function getTaskStreamUrl(id: string) {
  const token = localStorage.getItem('token')
  return `/api/v1/tasks/${id}/stream${token ? `?token=${token}` : ''}`
}
