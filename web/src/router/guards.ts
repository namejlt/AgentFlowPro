import router from './index'

router.beforeEach(async (to, from, next) => {
  const token = localStorage.getItem('token')

  if (to.meta.public) {
    if (token && to.path === '/login') {
      next('/dashboard')
      return
    }
    next()
    return
  }

  if (!token) {
    next('/login')
    return
  }

  next()
})
