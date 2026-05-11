import { computed } from 'vue'
import { useUserStore } from '@/stores/user'

export function usePermission() {
  const userStore = useUserStore()

  const isCreator = computed(() => userStore.isCreator())
  const isAdmin = computed(() => userStore.isAdmin())
  const canCreate = computed(() => isCreator.value || isAdmin.value)
  const canManageModels = computed(() => isCreator.value || isAdmin.value)
  const canManageUsers = computed(() => isAdmin.value)
  const canViewAudit = computed(() => isAdmin.value)
  const canManageSystem = computed(() => isAdmin.value)

  return { isCreator, isAdmin, canCreate, canManageModels, canManageUsers, canViewAudit, canManageSystem }
}
