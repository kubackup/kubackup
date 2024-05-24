import request from '@/utils/request'

export function fetchSearch(query) {
  return request({
    url: '/task',
    method: 'get',
    params: query
  })
}

/**
 * 立即执行备份
 * @param plan_id
 * @returns {AxiosPromise}
 */
export function fetchBackup(plan_id) {
  return request({
    url: `/task/backup/${plan_id}`,
    method: 'post'
  })
}

/**
 * 还原数据
 * @param repoid
 * @param snapid
 * @param data
 * @returns {AxiosPromise}
 */
export function fetchRestore(repoid, snapid, data) {
  return request({
    url: `/task/${repoid}/restore/${snapid}`,
    method: 'post',
    data
  })
}

