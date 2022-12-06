const AUTH_LOCAL_STORAGE_KEY: string = 'mt-auth'


/**
 * @description Get token from local storage
 */
export const getToken = (): string | null => {
  return window.localStorage.getItem(AUTH_LOCAL_STORAGE_KEY)
}

/**
 * @description Set token to local storage
 * @param data: string
 */
export const setToken = (data: string) => {
  window.localStorage.setItem(AUTH_LOCAL_STORAGE_KEY, data)
}

/**
 * @description Remove token from local storage
 */
export const removeToken = (): void => {
  window.localStorage.removeItem(AUTH_LOCAL_STORAGE_KEY)
}

export default { getToken, setToken, removeToken }