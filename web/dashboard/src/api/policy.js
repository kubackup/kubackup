import request from '@/utils/request'

export function fetchList(query) {
  return request({
    url: '/policy',
    method: 'get',
    params: query
  })
}

export function fetchCreate(data) {
  return request({
    url: '/policy',
    method: 'post',
    data
  })
}

export function fetchUpdate(data) {
  return request({
    url: `/policy/${data.id}`,
    method: 'put',
    data
  })
}

export function fetchDel(id) {
  return request({
    url: `/policy/${id}`,
    method: 'delete'
  })
}

export function fetchDoPolicy(id) {
  return request({
    url: `/policy/do/${id}`,
    method: 'post'
  })
}
