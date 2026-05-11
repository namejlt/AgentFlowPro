import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { UserItem } from '@/types'
import { getMe, login as apiLogin, logout as apiLogout } from '@/api/auth'
import router from '@/router'

export const useUserStore = defineStore('user', () => {
  const user = ref<UserItem | null>(null)
  const token = ref(localStorage.getItem('token') || '')

  async function login(email: string, password: string) {
    const res = await apiLogin({ email, password })
    token.value = res.data.data.token
    user.value = res.data.data.user
    localStorage.setItem('token', token.value)
  }

  async function fetchMe() {
    try {
      const res = await getMe()
      user.value = res.data.data
    } catch {
      user.value = null
      token.value = ''
      localStorage.removeItem('token')
    }
  }

  async function logout() {
    try {
      await apiLogout()
    } finally {
      user.value = null
      token.value = ''
      localStorage.removeItem('token')
      router.push('/login')
    }
  }

  function isCreator() {
    return user.value?.role === 'creator' || user.value?.role === 'admin'
  }

  function isAdmin() {
    return user.value?.role === 'admin'
  }

  return { user, token, login, fetchMe, logout, isCreator, isAdmin }
})
