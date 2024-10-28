import request from '@/utils/request'

export function fetchList(query) {
  return request({
    url: '/repository',
    method: 'get',
    params: query
  })
}

export function fetchGet(id) {
  return request({
    url: `/repository/${id}`,
    method: 'get'
  })
}

export function fetchDel(id) {
  return request({
    url: `/repository/${id}`,
    method: 'delete'
  })
}

export function fetchCreate(data) {
  return request({
    url: '/repository',
    method: 'post',
    data
  })
}

export function fetchUpdate(data) {
  return request({
    url: `/repository/${data.id}`,
    method: 'put',
    data
  })
}

export function fetchSnapshotsList(query) {
  return request({
    url: `/restic/${query.id}/snapshots`,
    method: 'get',
    params: query
  })
}

export function fetchLsList(repo, snap, query) {
  return request({
    url: `/restic/${repo}/ls/${snap}`,
    method: 'get',
    params: query
  })
}

export function fetchSearchList(repo, snap, query) {
  return request({
    url: `/restic/${repo}/search/${snap}`,
    method: 'get',
    params: query
  })
}

export function fetchParmsList(id) {
  return request({
    url: `/restic/${id}/parms`,
    method: 'get'
  })
}

export function fetchParmsMyList(id) {
  return request({
    url: `/restic/${id}/parmsForMy`,
    method: 'get'
  })
}

export function fetchLoadIndex(id) {
  return request({
    url: `/restic/${id}/loadIndex`,
    method: 'get'
  })
}

export function fetchCheck(repo) {
  return request({
    url: `/restic/${repo}/check`,
    method: 'post'
  })
}

export function fetchRebuildIndex(repo) {
  return request({
    url: `/restic/${repo}/rebuild-index`,
    method: 'post'
  })
}

export function fetchPrune(repo) {
  return request({
    url: `/restic/${repo}/prune`,
    method: 'post'
  })
}

/**
 * 升级数据格式版本
 * @param repo
 * @returns {*}
 */
export function fetchMigrate(repo) {
  return request({
    url: `/restic/${repo}/migrate`,
    method: 'post'
  })
}

/**
 * 删除快照
 * @param repo
 * @param snapshotid
 * @returns {AxiosPromise}
 */
export function fetchForget(repo, snapshotid) {
  return request({
    url: `/restic/${repo}/forget`,
    method: 'post',
    params: {
      snapshotid: snapshotid
    }
  })
}

/**
 * 解锁
 * @param repo
 * @param all
 * @returns {AxiosPromise}
 */
export function fetchUnlock(repo, all) {
  return request({
    url: `/restic/${repo}/unlock`,
    method: 'post',
    params: {
      all: all
    }
  })
}

export function fetchLastOper(repo, type) {
  return request({
    url: `/operation/last/${type}/${repo}`,
    method: 'get'
  })
}

