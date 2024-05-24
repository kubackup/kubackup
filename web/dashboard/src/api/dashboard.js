import request from '@/utils/request'

/**
 * 首页数据
 * @param query
 * @returns {AxiosPromise}
 */
export function fetchIndex(query) {
  return request({
    url: '/dashboard/index',
    method: 'get',
    params: query
  })
}

export function fetchDoGetAllRepoStats() {
  return request({
    url: '/dashboard/doGetAllRepoStats',
    method: 'post'
  })
}

/**
 * 查询操作日志
 * @param query
 * @returns {AxiosPromise}
 */
export function fetchLogs(query) {
  return request({
    url: '/dashboard/logs',
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
    url: '/version',
    method: 'get'
  })
}
