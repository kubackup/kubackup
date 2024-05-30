import {getInfo, login, refreshToken} from '@/api/user'
import {getToken, removeToken, removeUserInfo, setToken, setUserInfo} from '@/utils/auth'
import router, {resetRouter} from '@/router'

const state = {
  token: getToken(),
  name: '',
  introduction: '',
  roles: [],
  userinfo: {},
}

const mutations = {
  SET_TOKEN: (state, token) => {
    state.token = token
  },
  SET_INTRODUCTION: (state, introduction) => {
    state.introduction = introduction
  },
  SET_NAME: (state, name) => {
    state.name = name
  },
  SET_ROLES: (state, roles) => {
    state.roles = roles
  },
  SET_USERINFO: (state, userinfo) => {
    state.userinfo = userinfo
  }
}

const actions = {
  // user login
  login({commit}, userInfo) {
    const {username, password, code} = userInfo
    return new Promise((resolve, reject) => {
      login({username: username.trim(), password: password.trim(), code: code}).then(response => {
        const {data} = response
        if (data === null) {
          reject('mfa')
          return
        }
        const token = data.token
        if (token === '' || token === null) {
          reject('mfa')
          return
        }
        commit('SET_TOKEN', token)
        commit('SET_USERINFO', data)
        setToken(token)
        setUserInfo(data)
        resolve()
      }).catch(error => {
        reject(error)
      })
    })
  },

  // get user info
  getInfo({commit, state}) {
    return new Promise((resolve, reject) => {
      getInfo(state.token).then(response => {
        const {roles, name, introduction} = response

        // roles must be a non-empty array
        if (!roles || roles.length <= 0) {
          reject('getInfo: roles must be a non-null array!')
        }

        commit('SET_ROLES', roles)
        commit('SET_NAME', name)
        commit('SET_INTRODUCTION', introduction)
        resolve(response)
      }).catch(error => {
        reject(error)
      })
    })
  },

  // user logout
  logout({commit, state, dispatch}) {
    return new Promise((resolve, reject) => {
      commit('SET_TOKEN', '')
      commit('SET_ROLES', [])
      commit('SET_USERINFO', {})
      removeToken()
      removeUserInfo()
      resetRouter()

      // reset visited views and cached views
      // to fixed https://github.com/PanJiaChen/vue-element-admin/issues/2485
      dispatch('tagsView/delAllViews', null, {root: true})

      resolve()
    })
  },

  // refresh token
  refreshToken({commit}) {
    return new Promise(resolve => {
      refreshToken().then(res => {
        const {data} = res
        commit('SET_TOKEN', data)
        setToken(data)
        resolve()
      })
    })
  },

  // dynamically modify permissions
  async changeRoles({commit, dispatch}, role) {
    const token = role + '-token'

    commit('SET_TOKEN', token)
    setToken(token)

    const {roles} = await dispatch('getInfo')

    resetRouter()

    // generate accessible routes map based on roles
    const accessRoutes = await dispatch('permission/generateRoutes', roles, {root: true})
    // dynamically add accessible routes
    router.addRoutes(accessRoutes)

    // reset visited views and cached views
    dispatch('tagsView/delAllViews', null, {root: true})
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
