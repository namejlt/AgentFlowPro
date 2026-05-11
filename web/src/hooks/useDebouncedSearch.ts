import { ref } from 'vue'

export function useDebouncedSearch(delay = 300) {
  const keyword = ref('')
  let timer: ReturnType<typeof setTimeout> | null = null

  function onSearch(callback: (val: string) => void) {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      callback(keyword.value)
    }, delay)
  }

  return { keyword, onSearch }
}
