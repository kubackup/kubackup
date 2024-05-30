import request from '@/utils/request'

export function login(data) {
  return request({
    url: '/login',
    method: 'post',
    data
  })
}

/**
 * 刷新token
 * @returns {AxiosPromise}
 */
export function refreshToken() {
  return request({
    url: '/refreshToken',
    method: 'post'
  })
}

export function getInfo(token) {
  return Promise.resolve({
    roles: ['admin'],
    introduction: 'I am a super administrator',
    name: 'Super Admin'
  })
}

export function fetchList() {
  return request({
    url: '/user',
    method: 'get'
  })
}

export function fetchDel(id) {
  return request({
    url: `/user/${id}`,
    method: 'delete'
  })
}

export function fetchCreate(data) {
  return request({
    url: '/user',
    method: 'post',
    data
  })
}

export function fetchUpdate(data) {
  return request({
    url: `/user/${data.id}`,
    method: 'put',
    data
  })
}

export function fetchRePwd(data) {
  return request({
    url: '/repwd',
    method: 'post',
    data
  })
}

export function fetchOtp(){
  return request({
    url: '/otp',
    method: 'get',
  })
}

export function fetchBindOtp(data){
  return request({
    url: '/otp',
    method: 'post',
    data
  })
}

export function fetchDeleteOtp(){
  return request({
    url: '/otp',
    method: 'put',
  })
}
