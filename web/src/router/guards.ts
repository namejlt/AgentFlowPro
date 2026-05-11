import router from './index'
import { useUserStore } from '@/stores/user'

router.beforeEach(async (to, _from, next) => {
  const userStore = useUserStore()
  if (to.meta.public) {
    next()
    return
  }
  if (!userStore.token) {
    next('/login')
    return
  }
  if (!userStore.user) {
    await userStore.fetchMe()
    if (!userStore.user) {
      next('/login')
      return
    }
  }
  next()
})
