<template>
  <div class="app-container">
    <div>
      <el-row :gutter="40" class="panel-group">
        <el-col :xs="6" :sm="6" :lg="6" class="card-panel-col">
          <el-form label-position="right" label-width="80px">
            <el-form-item label="名称:">
              <span>{{ repoData.name }}</span>
            </el-form-item>
            <el-form-item label="服务器:">
              <span>{{ repoData.endPoint }}</span>
            </el-form-item>
            <el-form-item label="桶:" v-if="repoData.bucket">
              <span>{{ repoData.bucket }}</span>
            </el-form-item>
            <el-form-item label="创建时间:">
              <span>{{ dateFormat(repoData.createdAt) }}</span>
            </el-form-item>
            <el-form-item label="类型:">
              <span>{{ formatType(repoData.type) }}</span>
            </el-form-item>
            <el-form-item label="格式版本:">
              <span>{{ repoData.repositoryVersion }}</span>
              <el-button type="text" @click="migrationHandler()">升级版本</el-button>
            </el-form-item>
            <el-form-item label="连接状态:">
              <el-tag :type="formatStatus(repoData.status).color">
                {{ formatStatus(repoData.status).name }}
              </el-tag>
            </el-form-item>
            <el-form-item label="错误信息:" v-if="repoData.errmsg!==''">
              <span>{{ repoData.errmsg }}</span>
            </el-form-item>
            <el-form-item label="数据状态:">
              <div class="form-btn">
                <el-tag :type="formatStatus(checkObj.status).color">
                  {{ formatStatus(checkObj.status).name }}
                </el-tag>
                <el-button type="text" @click="checkHandler()">重新检测</el-button>
              </div>
            </el-form-item>
            <el-form-item label="重建索引:">
              <div class="form-btn">
                <el-tag :type="formatStatus(rebuildIndexObj.status).color">
                  {{ formatStatus(rebuildIndexObj.status).name }}
                </el-tag>
                <el-button type="text" @click="rebuildIndexHandler()">重新执行</el-button>
              </div>
            </el-form-item>
            <el-form-item label="清理无用数据:">
              <div class="form-btn">
                <el-tag :type="formatStatus(pruneObj.status).color">
                  {{ formatStatus(pruneObj.status).name }}
                </el-tag>
                <el-button type="text" @click="pruneHandler()">重新执行</el-button>
              </div>
            </el-form-item>
            <el-form-item label="清除锁:">
              <div class="form-btn">
                <el-button type="text" @click="unlockHandler()">执行</el-button>
              </div>
            </el-form-item>
          </el-form>
        </el-col>
        <el-col :xs="18" :sm="18" :lg="18" class="card-panel-col">
          <el-tabs v-model="activeName" tab-position="left" @tab-click="handleTabClick">
            <el-tab-pane label="数据状态" name="1">
              <Terminal title="日志" showHeader :init="checkObj.init"
                        :data="checkObj.logs"/>
            </el-tab-pane>
            <el-tab-pane label="重建索引" name="2">
              <Terminal title="日志" showHeader :init="rebuildIndexObj.init"
                        :data="rebuildIndexObj.logs"/>
            </el-tab-pane>
            <el-tab-pane label="清理数据" name="3">
              <Terminal title="日志" showHeader :init="pruneObj.init"
                        :data="pruneObj.logs"/>
            </el-tab-pane>
            <el-tab-pane label="版本升级" name="5">
              <Terminal title="日志" showHeader :init="migrateObj.init"
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
import {repoStatusList, repoTypeList} from "@/consts";

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
      activeName: '1',
      statusList: repoStatusList,
      typeList: repoTypeList
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
        this.$notify.success('成功清理' + res.data + '个锁')
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
      if (!this.sock) {
        this.sock = new SockJS(ws_log + '?token=' + token)
        this.sock.onopen = function () {
          that.sock.send(JSON.stringify({Id: id}))
          that.sock.onmessage = function (e) {
            const data = JSON.parse(e.data);
            if (data.message) {
              that.$notify({
                type: 'error',
                title: '错误',
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
        res = {code: 1, name: '获取中', color: 'info'}
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
