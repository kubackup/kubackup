import request from '@/utils/request'

export function fetchLs(query) {
  return request({
    url: '/system/ls',
    method: 'get',
    params: query
  })
}
