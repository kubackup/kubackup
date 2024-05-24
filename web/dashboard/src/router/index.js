import Vue from 'vue'
import Router from 'vue-router'
/* Layout */
import Layout from '@/layout'

Vue.use(Router)

/**
 * Note: sub-menu only appear when route children.length >= 1
 * Detail see: https://panjiachen.github.io/vue-element-admin-site/guide/essentials/router-and-nav.html
 *
 * hidden: true                   if set true, item will not show in the sidebar(default is false)
 * alwaysShow: true               if set true, will always show the root menu
 *                                if not set alwaysShow, when item has more than one children route,
 *                                it will becomes nested mode, otherwise not show the root menu
 * redirect: noRedirect           if set noRedirect will no redirect in the breadcrumb
 * name:'router-name'             the name is used by <keep-alive> (must set!!!)
 * meta : {
    roles: ['admin','editor']    control the page roles (you can set multiple roles)
    title: 'title'               the name show in sidebar and breadcrumb (recommend set)
    icon: 'svg-name'/'el-icon-x' the icon show in the sidebar
    noCache: true                if set true, the page will no be cached(default is false)
    affix: true                  if set true, the tag will affix in the tags-view
    breadcrumb: false            if set false, the item will hidden in breadcrumb(default is true)
    activeMenu: '/example/list'  if set path, the sidebar will highlight the path you set
  }
 */

/**
 * constantRoutes
 * a base page that does not have permission requirements
 * all roles can be accessed
 */
export const constantRoutes = [
  {
    path: '/redirect',
    component: Layout,
    hidden: true,
    children: [
      {
        path: '/redirect/:path(.*)',
        component: () => import('@/views/redirect/index')
      }
    ]
  },
  {
    path: '/login',
    component: () => import('@/views/login/index'),
    hidden: true
  },
  {
    path: '/404',
    component: () => import('@/views/error-page/404'),
    hidden: true
  },
  {
    path: '/403',
    component: () => import('@/views/error-page/403'),
    hidden: true
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        component: () => import('@/views/dashboard/index'),
        name: 'Dashboard',
        meta: { title: '首页', icon: 'dashboard', affix: true, noCache: true }
      }
    ]
  },
  {
    path: '/profile',
    component: Layout,
    redirect: '/profile/index',
    hidden: true,
    children: [
      {
        path: 'index',
        component: () => import('@/views/profile/index'),
        name: 'Profile',
        meta: { title: '个人中心', icon: 'user', noCache: true }
      }
    ]
  }
]

/**
 * asyncRoutes
 * the routes that need to be dynamically loaded based on user roles
 */
export const asyncRoutes = [
  {
    path: '/repository',
    component: Layout,
    redirect: '/repository/index',
    name: 'Repository',
    meta: {
      title: '',
      icon: 'el-icon-s-help'
    },
    children: [
      {
        path: 'index',
        component: () => import('@/views/repository/index'),
        name: 'RepositoryList',
        meta: { title: '存储库', noCache: true }
      },
      {
        path: 'restore/:id',
        component: () => import('@/views/repository/restore'),
        name: 'Restore',
        meta: { title: '恢复', activeMenu: '/repository/index' },
        hidden: true
      },
      {
        path: 'snapshot/:id',
        component: () => import('@/views/repository/snapshot'),
        name: 'Snapshot',
        meta: { title: '快照', activeMenu: '/repository/index' },
        hidden: true
      },
      {
        path: 'operation/:id',
        component: () => import('@/views/repository/operation'),
        name: 'Operation',
        meta: { title: '维护', activeMenu: '/repository/index' },
        hidden: true
      }
    ]
  },
  {
    path: '/Plan',
    component: Layout,
    redirect: 'noRedirect',
    name: 'Plan',
    meta: {
      title: '',
      icon: 'table'
    },
    children: [
      {
        path: 'index',
        component: () => import('@/views/plan/index'),
        name: 'PlanList',
        meta: { title: '备份计划', noCache: true }
      }
    ]
  },
  {
    path: '/Task',
    component: Layout,
    redirect: 'noRedirect',
    name: 'Task',
    meta: {
      title: '',
      icon: 'list'
    },
    children: [
      {
        path: 'index',
        component: () => import('@/views/task/index'),
        name: 'TaskList',
        meta: { title: '任务记录' }
      }
    ]
  },
  {
    path: '/user',
    component: Layout,
    redirect: 'noRedirect',
    name: 'User',
    meta: {
      title: '',
      icon: 'el-icon-user-solid'
    },
    children: [
      {
        path: 'index',
        component: () => import('@/views/user/index'),
        name: 'UserList',
        meta: { title: '用户管理' }
      }
    ]
  },
  {
    path: '/log',
    component: Layout,
    redirect: 'noRedirect',
    name: 'Log',
    meta: {
      title: '',
      icon: 'nested'
    },
    children: [
      {
        path: 'index',
        component: () => import('@/views/log/index'),
        name: 'LogList',
        meta: { title: '操作日志' }
      }
    ]
  },

  // 404 page must be placed at the end !!!
  { path: '*', redirect: '/404', hidden: true }
]

const createRouter = () => new Router({
  // mode: 'history', // require service support
  scrollBehavior: () => ({ y: 0 }),
  mode: 'history',
  routes: constantRoutes
})

const router = createRouter()

export function resetRouter() {
  const newRouter = createRouter()
  router.matcher = newRouter.matcher // reset router
}

export default router
