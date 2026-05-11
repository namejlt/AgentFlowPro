import http from './http'
import type { ApiResponse, ReportItem, PaginationParams } from '@/types'

export function getReports(params?: PaginationParams & { workflow_id?: string; archived?: boolean; status?: string }) {
  return http.get<ApiResponse<ReportItem[]>>('/api/v1/reports', { params })
}

export function getReport(id: string) {
  return http.get<ApiResponse<ReportItem>>(`/api/v1/reports/${id}`)
}

export function deleteReport(id: string) {
  return http.delete<ApiResponse<void>>(`/api/v1/reports/${id}`)
}

export function archiveReport(id: string, archived: boolean) {
  return http.patch<ApiResponse<void>>(`/api/v1/reports/${id}/archive`, { archived })
}

export function batchDeleteReports(ids: string[]) {
  return http.post<ApiResponse<void>>('/api/v1/reports/batch-delete', { ids })
}

export function exportReportMd(id: string) {
  return http.get(`/api/v1/reports/${id}/export/md`, { responseType: 'blob' })
}

export function exportReportPdf(id: string) {
  return http.get(`/api/v1/reports/${id}/export/pdf`, { responseType: 'blob' })
}

export function exportReportDocx(id: string) {
  return http.get(`/api/v1/reports/${id}/export/docx`, { responseType: 'blob' })
}
