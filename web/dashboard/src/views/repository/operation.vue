<template>
  <div class="app-container">
    <div>
      <el-row :gutter="40" class="panel-group">
        <el-col :xs="6" :sm="6" :lg="6" class="card-panel-col">
          <el-form label-position="right" label-width="80px">
            <el-form-item :label="$t('msg.name')+':'">
              <span>{{ repoData.name }}</span>
            </el-form-item>
            <el-form-item :label="$t('msg.endPoint')+':'">
              <span>{{ repoData.endPoint }}</span>
            </el-form-item>
            <el-form-item :label="$t('msg.bucket')+':'" v-if="repoData.bucket">
              <span>{{ repoData.bucket }}</span>
            </el-form-item>
            <el-form-item :label="$t('msg.createdAt')+':'">
              <span>{{ dateFormat(repoData.createdAt) }}</span>
            </el-form-item>
            <el-form-item :label="$t('msg.type')+':'">
              <span>{{ formatType(repoData.type) }}</span>
            </el-form-item>
            <el-form-item :label="$t('msg.repositoryVersion')+':'">
              <span>{{ repoData.repositoryVersion }}</span>
              <el-button type="text" @click="migrationHandler()">{{ $t('msg.operation.update') }}</el-button>
            </el-form-item>
            <el-form-item :label="$t('msg.statusData')+':'">
              <el-tag :type="formatStatus(repoData.status).color">
                {{ formatStatus(repoData.status).name }}
              </el-tag>
            </el-form-item>
            <el-form-item :label="$t('msg.title.errorInfo')+':'" v-if="repoData.errmsg!==''">
              <span>{{ repoData.errmsg }}</span>
            </el-form-item>
            <el-form-item :label="$t('msg.statusConn')+':'">
              <div class="form-btn">
                <el-tag :type="formatStatus(checkObj.status).color">
                  {{ formatStatus(checkObj.status).name }}
                </el-tag>
                <el-button type="text" @click="checkHandler()">{{ $t('msg.operation.check') }}</el-button>
              </div>
            </el-form-item>
            <el-form-item :label="$t('msg.operation.rebuildIndex')+':'">
              <div class="form-btn">
                <el-tag :type="formatStatus(rebuildIndexObj.status).color">
                  {{ formatStatus(rebuildIndexObj.status).name }}
                </el-tag>
                <el-button type="text" @click="rebuildIndexHandler()">{{ $t('msg.operation.execute') }}</el-button>
              </div>
            </el-form-item>
            <el-form-item :label="$t('msg.operation.prune')+':'">
              <div class="form-btn">
                <el-tag :type="formatStatus(pruneObj.status).color">
                  {{ formatStatus(pruneObj.status).name }}
                </el-tag>
                <el-button type="text" @click="pruneHandler()">{{ $t('msg.operation.execute') }}</el-button>
              </div>
            </el-form-item>
            <el-form-item :label="$t('msg.operation.unlock')+':'">
              <div class="form-btn">
                <el-button type="text" @click="unlockHandler()">{{ $t('msg.operation.execute') }}</el-button>
              </div>
            </el-form-item>
          </el-form>
        </el-col>
        <el-col :xs="18" :sm="18" :lg="18" class="card-panel-col">
          <el-tabs v-model="activeName" tab-position="left" @tab-click="handleTabClick">
            <el-tab-pane :label="$t('msg.statusData')" name="1">
              <Terminal :title="$t('msg.title.log')" showHeader :init="checkObj.init"
                        :data="checkObj.logs"/>
            </el-tab-pane>
            <el-tab-pane :label="$t('msg.operation.rebuildIndex')" name="2">
              <Terminal :title="$t('msg.title.log')" showHeader :init="rebuildIndexObj.init"
                        :data="rebuildIndexObj.logs"/>
            </el-tab-pane>
            <el-tab-pane :label="$t('msg.operation.prune')" name="3">
              <Terminal :title="$t('msg.title.log')" showHeader :init="pruneObj.init"
                        :data="pruneObj.logs"/>
            </el-tab-pane>
            <el-tab-pane :label="$t('msg.repositoryVersion')" name="5">
              <Terminal :title="$t('msg.title.log')" showHeader :init="migrateObj.init"
                        :data="migrateObj.logs"/>
            </el-tab-pane>

          </el-tabs>

        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script>
import {
  fetchCheck,
  fetchGet,
  fetchLastOper,
  fetchMigrate,
  fetchPrune,
  fetchRebuildIndex,
  fetchUnlock
} from '@/api/repository'
import Terminal from '@/components/TermLog'
import {dateFormat} from "@/utils";
import SockJS from 'sockjs-client'
import {getToken} from "@/utils/auth";
import {ws_log} from "@/api/ws";
import {repoStatusList, repoStatusListEN, repoTypeList, repoTypeListEN} from "@/consts";

export default {
  name: 'Operation',
  components: {
    Terminal
  },
  data() {
    return {
      checkObj: {
        status: 1,
        init: true,
        logs: []
      },
      rebuildIndexObj: {
        status: 1,
        init: true,
        logs: []
      },
      pruneObj: {
        status: 1,
        init: true,
        logs: []
      },
      migrateObj: {
        status: 1,
        init: true,
        logs: []
      },
      repoData: {
        type: 4
      },
      listQuery: {
        id: 0,
      },
      sock: null,
      curSockObj: 1,
      activeName: '1'
    }
  },
  computed: {
    statusList() {
      return this.$i18n.locale === 'zh-CN' ? repoStatusList : repoStatusListEN
    },
    typeList() {
      return this.$i18n.locale === 'zh-CN' ? repoTypeList : repoTypeListEN
    }
  },
  created() {
    this.listQuery.id = this.$route.params && this.$route.params.id
    this.getRepo()
    this.getLastOper(5)
    this.getLastOper(3)
    this.getLastOper(2)
    this.getLastOper(1)
  },
  beforeDestroy() {
    this.sock = null
  },
  methods: {
    checkHandler() {
      this.checkObj.logs = []
      this.checkObj.init = true
      fetchCheck(this.listQuery.id).then(res => {
        this.curSockObj = 1
        this.openSockjs(res.data)
      }).finally(() => {
        this.activeName = '1'
      })
    },
    migrationHandler() {
      this.migrateObj.logs = []
      this.migrateObj.init = true
      fetchMigrate(this.listQuery.id).then(res => {
        this.curSockObj = 5
        this.openSockjs(res.data)
      }).finally(() => {
        this.activeName = '5'
      })
    },
    rebuildIndexHandler() {
      this.rebuildIndexObj.logs = []
      this.rebuildIndexObj.init = true
      fetchRebuildIndex(this.listQuery.id).then(res => {
        this.curSockObj = 2
        this.openSockjs(res.data)
      }).finally(() => {
        this.activeName = '2'
      })
    },
    pruneHandler() {
      this.pruneObj.logs = []
      this.pruneObj.init = true
      fetchPrune(this.listQuery.id).then(res => {
        this.curSockObj = 3
        this.openSockjs(res.data)
      }).finally(() => {
        this.activeName = '3'
      })
    },
    unlockHandler() {
      fetchUnlock(this.listQuery.id).then(res => {
        this.$notify.success(this.$t('repository.unlockSuccess', {count: res.data}))
      })
    },
    getLastOper(type) {
      fetchLastOper(this.listQuery.id, type).then(res => {
        let info = res.data
        if (!info.logs) {
          info.logs = []
        }
        if (info.status !== 1) {
          // 仅保留info数据中的最新100条
          if (info.logs.length > 100) {
            info.logs = info.logs.slice(info.logs.length - 100)
          }
          this.updateLog(type, info, false)
        } else {
          this.curSockObj = type
          this.openSockjs(info.id)
        }
      })
    },
    updateLog(type, data, isPush) {
      switch (type) {
        case 1:
          if (isPush) {
            this.checkObj.logs.push(data)
          } else {
            this.checkObj = data
          }
          this.checkObj.init = !isPush
          break
        case 2:
          if (isPush) {
            this.rebuildIndexObj.logs.push(data)
          } else {
            this.rebuildIndexObj = data
          }
          this.rebuildIndexObj.init = !isPush
          break
        case 3:
          if (isPush) {
            this.pruneObj.logs.push(data)
          } else {
            this.pruneObj = data
          }
          this.pruneObj.init = !isPush
          break
        case 5:
          if (isPush) {
            this.migrateObj.logs.push(data)
          } else {
            this.migrateObj = data
          }
          this.migrateObj.init = !isPush
          break
        default:
      }
    },
    openSockjs(id) {
      const token = getToken().token
      const that = this
      if (this.sock === null) {
        this.sock = new SockJS(ws_log + '?token=' + token)
        this.sock.onopen = function () {
          that.sock.send(JSON.stringify({Id: id}))
          that.sock.onmessage = function (e) {
            const data = JSON.parse(e.data);
            if (data.message) {
              that.$notify({
                type: 'error',
                title: that.$t('common.error'),
                message: data.message
              })
              that.getLastOper(that.curSockObj)
            } else {
              that.updateLog(that.curSockObj, data, true)
            }
          }
        }
        this.sock.onclose = function () {
          that.sock = null
          that.getLastOper(that.curSockObj)
        }
      } else {
        this.sock.send(JSON.stringify({Id: id}))
      }
    },
    getRepo() {
      fetchGet(this.listQuery.id).then(res => {
        this.repoData = res.data
      })
    },
    handleTabClick() {
      this.getLastOper(Number(this.activeName))
    },
    formatStatus(code) {
      let res = this.statusList.find(item => item.code === code)
      if (!res) {
        res = {code: 1, name: this.$t('common.loading'), color: 'info'}
      }
      return res
    },
    dateFormat(cellValue) {
      return dateFormat(cellValue, 'yyyy-MM-dd hh:mm:ss')
    },
    formatType(code) {
      return this.typeList.find(item => item.code === code).name
    },
  }
}
</script>

<style lang="scss" scoped>
@import "src/styles/variables";

.tabs-content {
  background-color: $menuBg;
}

.form-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

</style>
