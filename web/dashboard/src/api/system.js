import request from '@/utils/request'

export function fetchLs(query) {
  return request({
    url: '/system/ls',
    method: 'get',
    params: query
  })
}

/**
 * 获取当前版本
 * @returns {AxiosPromise}
 */
export function fetchVersion() {
  return request({
    url: '/system/version',
    method: 'get'
  })
}

export function fetchLatestVersion() {
  return request({
    url: '/system/version/latest',
    method: 'get'
  })
}

export function fetchUpgradeVersion(data) {
  return request({
    url: '/system/upgradeVersion/'+data,
    method: 'post'
  })
}
