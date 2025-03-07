<template>
  <div class="app-container">
    <div class="handle-search">
      <el-form :model="listQuery" inline @submit.native.prevent>
        <el-form-item :label="$t('msg.select')">
          <el-select v-model="listQuery.path" clearable :placeholder="$t('msg.pleaseSelect')" @change="handleSearch">
            <el-option
              v-for="(item,id) in pathList"
              :key="id"
              :label="item"
              :value="item"
              style="width: 500px">
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-date-picker
            v-model="listQuery.date"
            type="date"
            :placeholder="$t('msg.pleaseSelect')"
            @change="handleSearch">
          </el-date-picker>
        </el-form-item>
      </el-form>
    </div>
    <div>
      <el-row :gutter="40" class="panel-group">
        <el-col :xs="8" :sm="8" :lg="8" class="card-panel-col">
          <el-card class="box-card">
            <div slot="header">
              <p>{{ $t('msg.host') }}：{{ hostname }}</p>
              <p>{{ $t('msg.path') }}：{{ listQuery.path }}</p>
              <p>{{ $t('msg.total') }}：{{ total }}</p>
              <div>
                <p>{{ $t('msg.cleanPolicy') }}：</p>
                <div v-if="policy.n > 0">
                  <p>{{ $t('msg.tips.keepLatestSnapshots') }} <i class="blue">{{ policy.n + formatType(policy.type).name }} </i> {{ $t('msg.snapshot') }} </p>
                  <el-button
                    class="forget-create"
                    type="text"
                    size="mini"
                    @click="handleUpdate()">
                    {{ $t('msg.operation.edit') }}
                  </el-button>
                  <el-button
                    class="forget-create"
                    type="text"
                    size="mini"
                    @click="handleDel()">
                    {{ $t('msg.operation.delete') }}
                  </el-button>
                  <el-button
                    class="forget-create"
                    type="text"
                    size="mini"
                    @click="doPolicy()">
                    {{ $t('msg.operation.execute') }}
                  </el-button>
                </div>
                <div v-else-if="policy.n === 0 && listQuery.path !== ''">
                  <el-button
                    class="forget-create"
                    type="text"
                    size="mini"
                    @click="handleAdd()">
                    {{ $t('msg.set') }}
                  </el-button>
                </div>
              </div>
            </div>
            <el-collapse v-model="activeName" accordion>
              <el-collapse-item :title="snaps.name" :name="snaps.name" :key="i" v-for="(snaps, i) in snapList">
                <el-timeline>
                  <el-timeline-item
                    v-for="(item, index) in snaps.list"
                    :key="index"
                    :timestamp="item.time|goDatToDateString"
                    type="primary"
                    icon="el-icon-success"
                    size="large"
                    placement="top"
                  >
                    <div class="timeline">
                      <span class="snap">{{ item.short_id }}</span>
                      <el-button
                        class="restore_btn"
                        type="text"
                        size="mini"
                        @click="deleteSnap(item.short_id)">
                        {{ $t('msg.operation.delete') }}
                      </el-button>
                    </div>
                  </el-timeline-item>
                </el-timeline>
              </el-collapse-item>
            </el-collapse>
            <p v-if="noMore" style="font-size: 20px; text-align: center; color: #bbbbbb">{{ $t('msg.tips.noMoreData') }}</p>
            <div v-else style="text-align: center;">
              <el-button :loading="listLoading" type="info" plain @click="getMoreList">{{ $t('msg.tips.loadMore') }}</el-button>
            </div>
          </el-card>
        </el-col>
        <el-col :xs="16" :sm="16" :lg="16" class="card-panel-col">
          <el-card class="box-card">
            <Terminal :title="$t('msg.logs')" showHeader :init="forgetObj.init"
                      :data="forgetObj.logs"/>
          </el-card>
        </el-col>
      </el-row>
    </div>
    <el-dialog :title="textMap[dialogStatus]" :visible.sync="dialogFormVisible" top="5vh" width="500px">
      <div>
        <el-input :placeholder="$t('msg.pleaseInput')" type="Number" v-model="temp.n" class="input-with-select" maxlength="2">
          <template slot="prepend">{{ $t('msg.keepLatest') }}：</template>
          <template slot="append">
            <el-select v-model="temp.type" :placeholder="$t('msg.pleaseSelect')" disabled>
              <el-option v-for="item in typeList" :key="item.code" :label="item.name" :value="item.code"/>
            </el-select>
          </template>
        </el-input>
        <p class="red">{{ $t('msg.systemWillKeep') }} {{ temp.n + formatType(temp.type).name }} {{ $t('msg.snapshots') }}，{{ $t('msg.exceedingPartsWillBeDeleted') }}</p>
      </div>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogFormVisible = false">
          {{ $t('msg.cancel') }}
        </el-button>
        <el-button
          type="primary"
          :loading="buttonLoading"
          @click=" dialogStatus === 'create' ? createPolicyData() : updatePolicyData()"
        >
          {{ $t('msg.confirm') }}
        </el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import {fetchForget, fetchLastOper, fetchParmsMyList, fetchSnapshotsList} from '@/api/repository'
import {dateFormat} from "@/utils";
import {getToken} from "@/utils/auth";
import Terminal from '@/components/TermLog'
import SockJS from "sockjs-client";
import {ws_log} from "@/api/ws";
import {ForgetTypeList, ForgetTypeListEN} from "@/consts";
import {fetchCreate, fetchDel, fetchDoPolicy, fetchList, fetchUpdate} from "@/api/policy";

export default {
  name: 'Snapshot',
  components: {
    Terminal
  },
  data() {
    return {
      typeList: this.$i18n.locale === 'zh-CN' ? ForgetTypeList : ForgetTypeListEN,
      list: [],
      total: 0,
      snapList: [],
      hostname: '',
      pathList: [],
      forgetObj: {
        status: 1,
        init: true,
        logs: []
      },
      textMap: {
        update: this.$t('msg.updateCleanupPolicy'),
        create: this.$t('msg.createCleanupPolicy')
      },
      dialogStatus: '',
      dialogFormVisible: false,
      buttonLoading: false,
      temp: {
        n: 90,
        type: 'last',
      },
      policy: {
        n: 0,
        type: '',
      },
      forget_type: [],
      sock: null,
      sockjsOpen: false,
      listLoading: false,
      noMore: true,
      activeName: '0',
      listQuery: {
        id: 0,
        path: '',
        date: '',
        host: '',
        pageNum: 1,
        pageSize: 100
      },
    }
  },
  created() {
    this.listQuery.id = this.$route.params && this.$route.params.id
    this.getParmList()
    this.getLastOper()
  },
  activated() {
    this.getParmList()
    this.getLastOper()
  },
  methods: {
    getMoreList() {
      this.listQuery.pageNum++
      this.getList()
    },
    handleSearch() {
      this.listQuery.host = this.hostname
      this.listQuery.pageNum = 1
      this.list = []
      this.snapList = []
      this.policy = {
        n: 0,
        type: '',
      }
      this.getList()
    },
    getParmList() {
      fetchParmsMyList(this.listQuery.id).then(res => {
        this.pathList = res.data.paths
        this.hostname = res.data.hostname
      })
    },
    formatType(code) {
      return this.typeList.find(item => item.code === code)
    },
    deleteSnap(snapid) {
      this.$confirm(this.$t('msg.confirmDeleteSnapshot') + '"' + snapid + '"？', this.$t('msg.deleteSnapshot'), {
        type: 'warning'
      }).then(() => {
        this.forgetObj.logs = []
        this.forgetObj.init = true
        fetchForget(this.listQuery.id, snapid).then(res => {
          this.openSockjs(res.data)
        })
      }).catch(() => {
      })
    },
    doPolicy() {
      this.$confirm(this.$t('msg.confirmExecuteCleanupPolicy'), this.$t('msg.executeCleanupPolicy'), {
        type: 'warning'
      }).then(() => {
        this.forgetObj.logs = []
        this.forgetObj.init = true
        fetchDoPolicy(this.policy.id).then(res => {
          this.openSockjs(res.data)
        })
      }).catch(() => {
      })
    },
    getLastOper() {
      fetchLastOper(this.listQuery.id, 4).then(res => {
        let info = res.data
        if (!info.logs) {
          info.logs = []
        }
        if (info.status !== 1) {
          // 仅保留info数据中的最新100条
          if (info.logs.length > 100) {
            info.logs = info.logs.slice(info.logs.length - 100)
          }
          this.updateLog(info, false)
        } else {
          this.openSockjs(info.id)
        }
      })
    },
    getList() {
      if (this.listQuery.path === '') {
        return
      }
      this.listLoading = true
      fetchSnapshotsList(this.listQuery).then(response => {
        this.list = this.list.concat(response.data.items)
        this.snapList = this.accordionList(this.list)
        this.noMore = Number(response.data.pageNum) * Number(response.data.pageSize) >= response.data.total
        this.total = response.data.total
        this.getPolicy()
      }).finally(() => {
        this.listLoading = false
      })
    },
    accordionList(list) {
      const res = [];
      list.forEach(l => {
        const name = dateFormat(l.time, 'yyyy-MM');
        if (this.listQuery.date) {
          const date = dateFormat(l.time, 'yyyy-MM-dd');
          const seldate = dateFormat(this.listQuery.date, 'yyyy-MM-dd');
          if (date !== seldate) {
            return
          }
          this.activeName = name
        }
        let find = false;
        res.forEach(r => {
          if (r.name === name) {
            r.list.push(l)
            find = true
          }
        })
        if (!find) {
          res.push({
            name: name,
            list: [l]
          })
        }
      })
      return res
    },
    handleAdd() {
      this.dialogStatus = 'create'
      this.dialogFormVisible = true
    },
    handleUpdate() {
      this.dialogStatus = 'update'
      this.dialogFormVisible = true
      this.temp = this.policy
    },
    handleDel() {
      fetchDel(this.policy.id).then(() => {
        this.getPolicy()
      })
    },
    updateLog(data, isPush) {
      if (isPush) {
        this.forgetObj.logs.push(data)
      } else {
        this.forgetObj = data
      }
      this.forgetObj.init = !isPush
    },
    openSockjs(id) {
      const token = getToken().token;
      const that = this;
      this.sock = new SockJS(ws_log + '?token=' + token)
      this.sock.onopen = function () {
        that.sockjsOpen = true
        that.sock.send(JSON.stringify({Id: id}))
        that.sock.onmessage = function (e) {
          const data = JSON.parse(e.data);
          if (data.message) {
            that.sockjsOpen = false
            that.$notify({
              type: 'error',
              title: '错误',
              message: data.message
            })
          } else {
            that.updateLog(data, true)
          }
        }
      }
      this.sock.onclose = function () {
        that.sockjsOpen = false
        that.handleSearch()
      }
    },
    getPolicy() {
      this.policy = {
        n: 0,
        type: '',
      }
      fetchList(this.listQuery).then(response => {
        let policys = response.data
        if (policys.length > 0) {
          this.policy.n = policys[0].value
          this.policy.type = policys[0].type
          this.policy.id = policys[0].id
        } else {
          this.policy.n = 0
          this.policy.type = ''
          this.policy.id = 0
        }
      })
    },
    createPolicyData() {
      let data = {
        repositoryId: Number(this.listQuery.id),
        path: this.listQuery.path,
        value: Number(this.temp.n),
        type: this.temp.type
      }
      fetchCreate(data).then(() => {
        this.dialogFormVisible = false
        this.getPolicy()
      })
    },
    updatePolicyData() {
      let data = {
        id: this.policy.id,
        repositoryId: Number(this.listQuery.id),
        path: this.listQuery.path,
        value: Number(this.temp.n),
        type: this.temp.type
      }
      fetchUpdate(data).then(() => {
        this.dialogFormVisible = false
        this.getPolicy()
      })
    }
  }
}
</script>

<style lang="scss" scoped>
@import "src/styles/variables";

.active {
  background: $light-blue;
  color: $menuHover;
}

.filenode {

}

.filenodes {
  margin-top: 10px;

  .confirmbtn {
    color: $menuHover;
  }
}

.red {
  color: red;
}

.blue {
  color: blue;
}

.filenode:hover {
  background-color: $light-blue;
  cursor: pointer;
  color: $menuHover;
}

.breadcrumb-item {
  padding: 5px;
}

.forget-create {
  font-size: 15px;
}

.breadcrumb-item:hover {
  background-color: $light-blue;
  cursor: pointer;
  color: $menuHover;

  .title {
    color: $menuHover;
  }
}

.input-with-select {
  background-color: #fff;

  .el-select {
    width: 110px;
  }
}

.file-tree {
  margin-top: 10px;
}

.timeline {
  width: 100%;
  justify-content: space-between;
  flex: 1;
  display: flex;
  align-items: center;

  .snap {
    padding: 10px 0 10px 0;
    width: 100%;
    font-size: 14px;
  }

  .restore_btn {
    padding: 15px;
    height: 100%;
  }

}

.panel-group {
  padding: 10px;

  .card-panel-col {
    background: #fff;
  }
}

.custom-tree-node {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;

  .file-title {
    font-size: 14px;
    padding-right: 8px;
  }
}
</style>
