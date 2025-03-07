import axios from 'axios'
import { Notification, Loading } from 'element-ui'
import store from '@/store'
import { getToken } from '@/utils/auth'
import router from '@/router'
import { fetchVersion } from '@/api/system'
import { i18n } from '@/i18n'

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

    // 添加语言设置到请求头
    config.headers['Accept-Language'] = i18n.locale

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

let RepoLoading = 'normal'

let timei = null

let loading = null

const getLoadingText = (load) => {
  switch (load) {
    case 'normal':
      return i18n.t('msg.common.normal')
    case 'loading':
      return i18n.t('msg.repository.loading')
    case 'upgrading':
      return i18n.t('msg.system.upgrading')
    default:
      return i18n.t('msg.common.normal')
  }
}

const checkRepoLoading = () => {
  if (RepoLoading !== 'normal') {
    if (loading != null) {
      return
    }
    loading = Loading.service({
      lock: true,
      text: getLoadingText(RepoLoading),
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
    if (timei != null) {
      clearInterval(timei)
    }
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
          title: i18n.t('msg.common.error'),
          message: res.message || i18n.t('msg.error.serverError'),
          type: 'error'
        })
        return Promise.reject(new Error(res.message || i18n.t('msg.error.serverError')))
      }
    } else {
      // 如果响应中包含语言设置，更新当前语言
      if (res.lang && res.lang !== i18n.locale) {
        i18n.locale = res.lang
        localStorage.setItem('locale', res.lang)
      }

      RepoLoading = res.systemStatus
      checkRepoLoading()
      return res
    }
  },
  error => {
    Notification({
      title: i18n.t('msg.common.error'),
      message: error.message,
      type: 'error'
    })

    // 处理HTTP状态码403的情况
    if (error.response && error.response.status === 403) {
      router.push('/403')
    }

    return Promise.reject(error)
  }
)

export default service
