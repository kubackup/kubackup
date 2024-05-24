import axios from 'axios'
import { Notification, Loading } from 'element-ui'
import store from '@/store'
import { getToken } from '@/utils/auth'
import router from '@/router'
import { fetchVersion } from '@/api/dashboard'

// create an axios instance
const service = axios.create({
  baseURL: process.env.VUE_APP_BASE_API, // url = base url + request url
  timeout: 300 * 1000 // request timeout
})

// request interceptor
service.interceptors.request.use(
  config => {
    if (!config.url.includes('/login')) {
      refreshToken()
      if (store.getters.token.token) {
        config.headers['Authorization'] = 'Bearer ' + getToken().token
      }
    }
    return config
  },
  error => {
    console.error(error) // for debug
    return Promise.reject(error)
  }
)
// 验证当前token是否过期
const isTokenExpired = () => {
  const expireTime = new Date(getToken().expiresAt).getTime() / 1000
  if (expireTime) {
    const nowTime = new Date().getTime() / 1000
    // 如果20分钟内将过期重新获取token
    return (expireTime - nowTime) < 1200
  }
  return false
}

let isRefreshing = false

const refreshToken = async() => {
  if (isTokenExpired() && !isRefreshing) {
    isRefreshing = true
    await store.dispatch('user/refreshToken').finally(() => {
      isRefreshing = false
    })
  }
}

let RepoLoading = false

let timei = null

let loading = null

const checkRepoLoading = () => {
  if (RepoLoading) {
    if (loading != null) {
      return
    }
    loading = Loading.service({
      lock: true,
      text: '仓库正在加载中，请稍后...',
      spinner: 'el-icon-loading',
      background: 'rgba(0, 0, 0, 0.7)'
    })
    timei = setInterval(() => {
      fetchVersion()
    }, 1000)
  } else {
    if (loading != null) {
      loading.close()
      location.reload()
    }
    if (timei != null) { clearInterval(timei) }
  }
}

// response interceptor
service.interceptors.response.use(
  /**
   * If you want to get http information such as headers or status
   * Please return  response => response
   */

  /**
   * Determine the request status by custom code
   * Here is just an example
   * You can also judge the status by HTTP Status Code
   */
  response => {
    const res = response.data
    if (!res.success) {
      if (res.code === 401) {
        store.dispatch('user/logout')
        router.push('/login')
      } else if (res.code === 403) {
        router.push('/403')
      } else {
        Notification({
          title: '错误',
          message: res.message || 'Error',
          type: 'error'
        })
        return Promise.reject(new Error(res.message || 'Error'))
      }
    } else {
      RepoLoading = !res.systemStatus
      checkRepoLoading()
      return res
    }
  },
  error => {
    Notification({
      title: '错误',
      message: error.message,
      type: 'error'
    })
    return Promise.reject(error)
  }
)

export default service
