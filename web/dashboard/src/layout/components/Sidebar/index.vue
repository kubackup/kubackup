<template>
  <div :class="{'has-logo':showLogo}">
    <logo v-if="showLogo" :collapse="isCollapse" />
    <el-scrollbar wrap-class="scrollbar-wrapper">
      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapse"
        :background-color="variables.menuBg"
        :text-color="variables.menuText"
        :unique-opened="false"
        :active-text-color="variables.menuActiveText"
        :collapse-transition="false"
        mode="horizontal"
      >
        <sidebar-item v-for="route in permission_routes" :key="route.path" :item="route" :base-path="route.path" />
      </el-menu>
      <div class="footer">
        <p>{{ version }}</p>
        <p>
          <el-link v-if="latestVersion" type="primary" :underline="false" @click="handleDialog">新版本：{{
            latestVersion
          }}
          </el-link>
        </p>
      </div>
    </el-scrollbar>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'
import Logo from './Logo'
import SidebarItem from './SidebarItem'
import variables from '@/styles/variables.scss'
import { fetchLatestVersion, fetchUpgradeVersion, fetchVersion } from '@/api/system'

export default {
  components: { SidebarItem, Logo },
  data() {
    return {
      version: '',
      latestVersion: ''
    }
  },
  computed: {
    ...mapGetters([
      'permission_routes',
      'sidebar'
    ]),
    activeMenu() {
      const route = this.$route
      const { meta, path } = route
      // if set path, the sidebar will highlight the path you set
      if (meta.activeMenu) {
        return meta.activeMenu
      }
      return path
    },
    showLogo() {
      return this.$store.state.settings.sidebarLogo
    },
    variables() {
      return variables
    },
    isCollapse() {
      return !this.sidebar.opened
    }
  },
  created() {
    this.getVersion()
  },
  methods: {
    getVersion() {
      fetchVersion().then(res => {
        const v = res.data
        if (v != null) {
          this.version = v.version
        }
        this.getLatestVersion()
      })
    },
    getLatestVersion() {
      fetchLatestVersion().then(res => {
        const version = res.data
        if (this.version !== version) {
          this.latestVersion = version
        }
      })
    },
    handleDialog() {
      this.$confirm('<a style="color: #3b91b6" href="https://kubackup.cn/changelog/" target="_blank">' + this.latestVersion + '更新日志</a>', '发现新版本', {
        dangerouslyUseHTMLString: true,
        confirmButtonText: '立即更新',
        cancelButtonText: '取消'
      }).then(() => {
        fetchUpgradeVersion(this.latestVersion).then(() => {
          this.getVersion()
        })
      })
    }
  }
}
</script>

<style lang="scss" scoped>
@import "../dashboard/src/styles/variables";

.footer {
  font-size: 15px;
  position: absolute;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  text-align: center;
}
</style>
