import Cookies from 'js-cookie'

const TokenKey = 'Token'
const UserInfo = 'UserInfo'

export function getToken() {
  const tokenstr = Cookies.get(TokenKey)
  if (tokenstr) {
    const tokeninfo = JSON.parse(tokenstr)
    if (tokeninfo === null) {
      return {
        token: undefined,
        expiresAt: undefined
      }
    }
    return tokeninfo
  } else {
    return {
      token: undefined,
      expiresAt: undefined
    }
  }
}

export function setToken(token) {
  return Cookies.set(TokenKey, token)
}

export function removeToken() {
  return Cookies.remove(TokenKey)
}

export function getUserInfo() {
  const str = Cookies.get(UserInfo)
  return JSON.parse(str)
}

export function setUserInfo(userinfo) {
  return Cookies.set(UserInfo, userinfo)
}

export function removeUserInfo() {
  return Cookies.remove(UserInfo)
}
