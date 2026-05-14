import axios from 'axios'
import type { ApiResponse } from '@/types'
import { ElMessage } from 'element-plus'
import router from '@/router'

const http = axios.create({
  baseURL: '',
  timeout: 300000,
  headers: { 'Content-Type': 'application/json' },
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    const data = response.data as ApiResponse
    if (data.code !== 0) {
      if (data.code === 1002) {
        localStorage.removeItem('token')
        router.push('/login')
      }
      const err = new Error(data.message || '请求失败') as any
      err.handled = true
      return Promise.reject(err)
    }
    return response
  },
  (error) => {
    if (error.handled) {
      return Promise.reject(error)
    }
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      router.push('/login')
      return Promise.reject(error)
    }
    return Promise.reject(error)
  }
)

export default http
