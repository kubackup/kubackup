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

export function fetchDumpFile(repo, snap, data) {
  return request({
    url: `/restic/${repo}/dump/${snap}`,
    method: 'post',
    data
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

export function fetchLastOper(repo, type) {
  return request({
    url: `/operation/last/${type}/${repo}`,
    method: 'get'
  })
}

