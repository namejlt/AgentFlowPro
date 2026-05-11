import http from './http'
import type { ApiResponse, LoginRequest, LoginResponse, UserItem, PaginationParams } from '@/types'

export function login(data: LoginRequest) {
  return http.post<ApiResponse<LoginResponse>>('/api/v1/auth/login', data)
}

export function logout() {
  return http.post<ApiResponse<void>>('/api/v1/auth/logout')
}

export function refreshToken() {
  return http.post<ApiResponse<{ token: string }>>('/api/v1/auth/refresh')
}

export function getMe() {
  return http.get<ApiResponse<UserItem>>('/api/v1/auth/me')
}

export function getUsers(params?: PaginationParams & { role?: string }) {
  return http.get<ApiResponse<UserItem[]>>('/api/v1/users', { params })
}

export function createUser(data: Partial<UserItem> & { password: string }) {
  return http.post<ApiResponse<UserItem>>('/api/v1/users', data)
}

export function getUser(id: string) {
  return http.get<ApiResponse<UserItem>>(`/api/v1/users/${id}`)
}

export function updateUser(id: string, data: Partial<UserItem>) {
  return http.patch<ApiResponse<UserItem>>(`/api/v1/users/${id}`, data)
}
