import router from './index'
import { getMe } from '@/api/auth'
import { ElMessage } from 'element-plus'

// Route guard: checks authentication and role-based access
router.beforeEach(async (to, from, next) => {
  const token = localStorage.getItem('token')

  // Public routes (login, register, etc.)
  if (to.meta.public) {
    if (token && to.path === '/login') {
      next('/dashboard')
      return
    }
    next()
    return
  }

  // No token - redirect to login
  if (!token) {
    next('/login')
    return
  }

  // Check token validity by fetching user info
  try {
    const res = await getMe()
    const user = res.data.data
    if (!user) {
      localStorage.removeItem('token')
      next('/login')
      return
    }

    // Store user info for permission checks
    localStorage.setItem('user_role', user.role || 'user')

    // Role-based access control
    const requiredRole = to.meta.role as string
    if (requiredRole && user.role !== requiredRole && user.role !== 'admin') {
      ElMessage.error('无权访问该页面')
      next('/dashboard')
      return
    }

    next()
  } catch {
    localStorage.removeItem('token')
    next('/login')
  }
})
