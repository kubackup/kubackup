import request from '@/utils/request'

export function fetchCreate(data) {
  return request({
    url: '/plan',
    method: 'post',
    data
  })
}

export function fetchUpdate(data) {
  return request({
    url: `/plan/${data.id}`,
    method: 'put',
    data
  })
}

export function fetchDel(id) {
  return request({
    url: `/plan/${id}`,
    method: 'delete'
  })
}

export function fetchList(query) {
  return request({
    url: '/plan',
    method: 'get',
    params: query
  })
}

export function fetchNextTime(query) {
  return request({
    url: '/plan/next_time',
    method: 'get',
    params: query
  })
}
