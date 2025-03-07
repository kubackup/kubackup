<template>
  <div class="app-container">
    <div class="handle-search">
      <el-form :model="listQuery" inline @submit.native.prevent>
        <el-form-item :label="$t('msg.plan')">
          <el-select v-model="listQuery.planId" class="handle-select mr5" clearable :placeholder="$t('msg.pleaseSelect')">
            <el-option
              v-for="(item, index) in [{id: 0, name: $t('msg.all')}].concat(planList)"
              :key="index"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('msg.name')">
          <el-input v-model="listQuery.name" placeholder="name" style="width: 150px;" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item :label="$t('msg.statusLabel')">
          <el-select v-model="listQuery.status" class="handle-select mr5" clearable :placeholder="$t('msg.pleaseSelect')">
            <el-option
              v-for="(item, index) in [{status: -1, name: $t('msg.all')}].concat(statusList)"
              :key="index"
              :label="item.name"
              :value="item.status"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="el-icon-search" @click="handleFilter">{{ $t('msg.search') }}</el-button>
        </el-form-item>
      </el-form>
    </div>
    <el-table :data="list" border fit highlight-current-row style="width: 100%">
      <el-table-column align="center" label="ID" width="80">
        <template slot-scope="scope">
          <span>{{ scope.row.id }}</span>
        </template>
      </el-table-column>

      <el-table-column prop="name" align="center" :label="$t('msg.name')"/>

      <el-table-column prop="createdAt" :formatter="dateFormat" align="center" :label="$t('msg.createdAt')"/>

      <el-table-column prop="duration" align="center" :label="$t('msg.duration')">
        <template slot-scope="{row}">
          <div>
            {{ (row.summary.totalDuration || row.progress.secondsElapsed) + '(' + (row.scanner.duration || '0:0') + ')' }}
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="dataAdd" align="center" :label="$t('msg.data')">
        <template slot-scope="{row}">
          {{ row.summary.dataAdded || row.progress.bytesDone }}
        </template>
      </el-table-column>
      <el-table-column prop="path" align="center" :label="$t('msg.path')"/>

      <el-table-column prop="repositoryId" align="center" :formatter="filterRepo" :label="$t('msg.repository')"/>

      <el-table-column prop="planId" align="center" :formatter="filterPlan" :label="$t('msg.plan')"/>

      <el-table-column align="center" :label="$t('msg.progress')">
        <template slot-scope="{row}">
          <el-progress
            type="dashboard"
            :width="50"
            :percentage="row.progress.percentDone?Number((row.progress.percentDone*100).toFixed(0)):100"
          />
        </template>
      </el-table-column>

      <el-table-column prop="status" align="center" :label="$t('msg.statusLabel')">
        <template slot-scope="{row}">
          <el-button :type="formatStatus(row.status).color" @click="handleInfo(row)">
            {{ formatStatus(row.status).name }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <pagination
      v-show="total > 0"
      :total="total"
      :page.sync="listQuery.pageNum"
      :limit.sync="listQuery.pageSize"
      :autoScroll="false"
      @pagination="getList"
    />

    <el-dialog v-if="taskType(taskInfo.name) === 1"
               :title="$t('msg.taskInfo')+' ' + (taskInfo.summary.snapshotId===undefined?'...':taskInfo.summary.snapshotId)"
               :visible.sync="dialogFormVisible" top="5vh"
               @close="closeSockjs"
               width="80%">
      <el-row :gutter="10">
        <el-col v-if="taskInfo.scanner && taskInfo.status!==3" :span="8">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title bluetext">{{ $t('msg.title.scannerInfo') }}</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="24"><p>{{ $t('msg.title.dataSize') }}：{{ taskInfo.scanner.dataSize }}</p></el-col>
              <el-col :span="24"><p>{{ $t('msg.title.totalFiles') }}：{{ taskInfo.scanner.totalFiles }}</p></el-col>
              <el-col :span="24"><p class="redtext">{{ $t('msg.title.scannerDuration') }}：{{ taskInfo.scanner.duration }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.status===3 && taskInfo.scannerError" :span="8">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title redtext">{{ $t('msg.title.scannerError') }}</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="24"><p>{{ $t('msg.title.errorInfo') }}：{{ taskInfo.scannerError.error }}</p></el-col>
              <el-col :span="24"><p>{{ $t('msg.title.ErrorItem') }}：{{ taskInfo.scannerError.item }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.summary && taskInfo.status!==3" :span="16">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title bluetext">{{ $t('msg.title.restoreInfo') }}</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="8"><p>
                {{ $t('msg.title.bytesDone') }}：{{ taskInfo.progress.bytesDone ? taskInfo.progress.bytesDone : taskInfo.scanner.dataSize }}</p>
              </el-col>
              <el-col :span="8"><p>{{ $t('msg.title.filesDone') }}：{{ taskInfo.progress.filesDone }}</p></el-col>
              <el-col v-if="taskInfo.progress.errorCount>0" :span="8" class="redtext"><p>
                {{ $t('msg.title.errorCount') }}：{{ taskInfo.progress.errorCount }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p class="redtext">{{ $t('msg.title.secondsElapsed') }}：{{ taskInfo.progress.secondsElapsed }}</p></el-col>
              <el-col v-if="taskInfo.progress.secondsRemaining" :span="8"><p class="redtext">
                {{ $t('msg.title.secondsRemaining') }}：{{ taskInfo.progress.secondsRemaining }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p>{{ $t('msg.title.avgSpeed') }}：{{ taskInfo.progress.avgSpeed }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.status===3 && (taskInfo.archivalError||taskInfo.restoreError)" :span="16">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title redtext">{{ $t('msg.title.restoreError') }}</span>
            </div>
            <el-row :gutter="10">
              <el-col v-if="taskInfo.restoreError" v-for="(item,index) in taskInfo.restoreError" :key="index"
                      :span="24"><p>
                {{ item.error + '-' + item.item }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
      </el-row>
      <el-progress
        style="margin-top: 10px"
        :percentage="taskInfo.progress.percentDone?Number((taskInfo.progress.percentDone*100).toFixed(0)):0"
      />
    </el-dialog>

    <el-dialog v-if="taskType(taskInfo.name) === 2"
               :title="$t('msg.taskInfo')+ ' '+(taskInfo.summary.snapshotId===undefined?'...':taskInfo.summary.snapshotId)"
               :visible.sync="dialogFormVisible" top="5vh"
               width="80%" @close="closeSockjs">
      <el-row :gutter="10">
        <el-col v-if="taskInfo.scanner && taskInfo.status!==3" :span="8">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title bluetext">{{ $t('msg.title.scannerInfo') }}</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="24"><p>{{ $t('msg.title.dataSize') }}：{{ taskInfo.scanner.dataSize }}</p></el-col>
              <el-col :span="24"><p>{{ $t('msg.title.totalFiles') }}：{{ taskInfo.scanner.totalFiles }}</p></el-col>
              <el-col :span="24"><p class="redtext">{{ $t('msg.title.scannerDuration') }}：{{ taskInfo.scanner.duration }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.status===3 && taskInfo.scannerError" :span="12">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title redtext">{{ $t('msg.title.scannerError') }}</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="24"><p>{{ $t('msg.title.errorInfo') }}：{{ taskInfo.scannerError.error }}</p></el-col>
              <el-col :span="24"><p>{{ $t('msg.title.ErrorItem') }}：{{ taskInfo.scannerError.item }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.status!==3" :span="16">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title bluetext">{{ $t('msg.title.summary') }}</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="8"><p>
                {{ $t('msg.title.bytesDone') }}：{{ taskInfo.progress.bytesDone ? taskInfo.progress.bytesDone : taskInfo.scanner.dataSize }}</p>
              </el-col>
              <el-col :span="8"><p>{{ $t('msg.title.dirsNew') }}：{{ taskInfo.summary.dirsNew }}</p></el-col>
              <el-col :span="8"><p>{{ $t('msg.title.filesNew') }}：{{ taskInfo.summary.filesNew }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p class="redtext">{{ $t('msg.title.dataAdded') }}：{{ taskInfo.summary.dataAdded }}</p></el-col>
              <el-col :span="8"><p>{{ $t('msg.title.dirsChanged') }}：{{ taskInfo.summary.dirsChanged }}</p></el-col>
              <el-col :span="8"><p>{{ $t('msg.title.filesChanged') }}：{{ taskInfo.summary.filesChanged }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p class="redtext">{{ $t('msg.title.filesDone') }}：{{ taskInfo.progress.filesDone }}</p></el-col>
              <el-col :span="8"><p>{{ $t('msg.title.dirsUnmodified') }}：{{ taskInfo.summary.dirsUnmodified }}</p></el-col>
              <el-col :span="8"><p>{{ $t('msg.title.filesUnmodified') }}：{{ taskInfo.summary.filesUnmodified }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p class="redtext">{{ $t('msg.title.secondsElapsed') }}：{{ taskInfo.progress.secondsElapsed }}</p></el-col>
              <el-col v-if="taskInfo.progress.secondsRemaining && sockjsOpen" :span="8"><p class="redtext">
                {{ $t('msg.title.secondsRemaining') }}：{{ taskInfo.progress.secondsRemaining }}</p></el-col>
              <el-col v-if="taskInfo.progress.errorCount>0" :span="8" class="redtext"><p>
                {{ $t('msg.title.errorCount') }}：{{ taskInfo.progress.errorCount }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p>{{ $t('msg.title.avgSpeed') }}：{{ taskInfo.progress.avgSpeed }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.archivalError" :span="12">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title redtext">{{ $t('msg.title.backupError') }}</span>
            </div>
            <el-col v-if="taskInfo.archivalError" v-for="(item,index) in taskInfo.archivalError" :key="index"
                    :span="24"><p>
              {{ item.error + '-' + item.item }}</p></el-col>
          </el-card>
        </el-col>
      </el-row>
      <el-row v-if="taskInfo.progress" style="margin-top: 10px">
        <el-col :span="24">
          <el-card>
            <div slot="header" class="clearfix">
              <span class="info_title">{{ $t('msg.progress') }}</span>
            </div>
            <div>
              <el-progress
                style="margin-top: 10px"
                :percentage="taskInfo.progress.percentDone?Number((taskInfo.progress.percentDone*100).toFixed(0)):0"
              />
            </div>
            <div>
              <p v-if="taskInfo.filelog" v-for="item in taskInfo.filelog">{{ item }}</p>
              <p v-if="taskInfo.progress.currentFiles" v-for="item in taskInfo.progress.currentFiles">{{ item }}</p>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </el-dialog>
  </div>
</template>

<script>
import {fetchSearch} from '@/api/task'
import {fetchList} from '@/api/repository'
import {fetchList as planFetchList} from '@/api/plan'
import Pagination from '@/components/Pagination'
import {dateFormat} from '@/utils'
import {getToken} from '@/utils/auth'
import SockJS from 'sockjs-client'
import {ws_task} from "@/api/ws";

export default {
  name: 'Task',
  components: {Pagination},
  data() {
    return {
      dialogFormVisible: false,
      statusList: [
        {name: this.$t('msg.new'), status: 0, color: 'primary'},
        {name: this.$t('msg.run'), status: 1, color: 'primary'},
        {name: this.$t('msg.completed'), status: 2, color: 'success'},
        {name: this.$t('msg.err'), status: 3, color: 'danger'}
      ],
      taskInfo: {
        summary: {},
        scanner: {},
        progress: {},
        filelog: []
      },
      repositoryList: [],
      planList: [],
      list: [],
      total: 0,
      sockjsOpen: false,
      hasRunning: false,
      sock: null,
      timei: null,
      listQuery: {
        name: '',
        status: -1,
        planId: 0,
        pageNum: 1,
        pageSize: 10
      }
    }
  },
  created() {
    this.getList()
    var that = this
    this.timei = setInterval(() => {
      if (that.hasRunning && !that.sockjsOpen) {
        that.getList()
      }
    }, 1000)
    fetchList().then(response => {
      this.repositoryList = response.data
    })
    planFetchList().then(res => {
      this.planList = res.data.items
    })
  },
  beforeDestroy() {
    clearInterval(this.timei)
  },
  methods: {
    taskType(name) {
      if (!name) {
        return 0
      }
      if (name.startsWith('restore')) {
        return 1
      } else if (name.startsWith('backup')) {
        return 2
      }
    },
    dateFormat(row, column, cellValue, index) {
      return dateFormat(cellValue, 'yyyy-MM-dd hh:mm')
    },
    formatStatus(code) {
      return this.statusList.find(item => item.status === code)
    },
    handleInfo(row) {
      this.dialogFormVisible = true
      this.resetTaskInfo()
      this.taskInfo = this.nullTaskInfo([row])[0]
      if (row.status === 1) {
        this.openSockjs(row.id)
      }
    },
    nullTaskInfo(taskInfos) {
      var has = false
      taskInfos.forEach(taskInfo => {
        if (taskInfo.status === 1) {
          has = true
        }
        if (taskInfo.summary === null || taskInfo.summary === undefined) {
          taskInfo.summary = {}
        }
        if (taskInfo.progress === null || taskInfo.progress === undefined) {
          taskInfo.progress = {}
        }
        if (taskInfo.scanner === null || taskInfo.scanner === undefined) {
          taskInfo.scanner = {}
        }
      })
      this.hasRunning = has
      return taskInfos
    },
    resetTaskInfo() {
      this.taskInfo = {
        summary: {},
        scanner: {},
        progress: {}
      }
    },
    handleFilter() {
      this.listQuery.pageNum = 1
      this.getList()
    },
    filterRepo(row, column, cellValue, index) {
      let res = cellValue
      this.repositoryList.forEach(value => {
        if (value.id === cellValue) {
          res = value.name
          return res
        }
      })
      return res
    },
    filterPlan(row, column, cellValue, index) {
      let res = cellValue
      this.planList.forEach(value => {
        if (value.id === cellValue) {
          res = value.name
          return res
        }
      })
      return res
    },
    openSockjs(id) {
      var token = getToken().token
      var that = this
      this.sock = new SockJS(ws_task + '?token=' + token)
      this.sock.onopen = function () {
        that.sockjsOpen = true
        that.sock.send(JSON.stringify({Id: Number(id)}))
        that.sock.onmessage = function (e) {
          var data = JSON.parse(e.data)
          if (data.message) {
            that.sockjsOpen = false
            that.$notify({
              type: 'error',
              title: this.$t('msg.err'),
              message: data.message
            })
          } else {
            that.updateProgress(data)
          }
        }
      }
      this.sock.onclose = function () {
        that.sockjsOpen = false
      }
    },
    closeSockjs() {
      if (this.sock) {
        this.sock.close()
        this.sock = null
      }
    },
    updateProgress(msg) {
      if (msg.messageType) {
        switch (msg.messageType) {
          case 'status':
            this.taskInfo.progress = msg
            break
          case 'error':
            this.$notify.error(msg.during + '\n' + msg.error + '\n' + msg.item)
            break
          case 'verbose_status':
            let log
            switch (msg.action) {
              case 'new':
                log = 'new       ' + msg.item + ', saved in ' + msg.duration + ' (' + msg.dataSize + ' added)'
                break
              case 'unchanged':
                log = 'unchanged ' + msg.item
                break
              case 'modified':
                log = 'modified  ' + msg.item + ', saved in ' + msg.duration + ' (' + msg.dataSize + ' added'
                break
              case 'scan_finished':
                this.taskInfo.scanner.duration = msg.duration
                this.taskInfo.scanner.totalFiles = msg.totalFiles
                this.taskInfo.scanner.dataSize = msg.dataSize
                break
            }
            this.taskInfo.filelog = [log]
            break
          case 'summary':
            this.taskInfo.summary = msg
            this.taskInfo.progress.percentDone = 1
            break
        }
      } else {
        console.info(msg)
        this.$notify.error(msg)
      }
    },
    getList() {
      fetchSearch(this.listQuery).then(response => {
        this.list = this.nullTaskInfo(response.data.items)
        this.total = response.data.total
      })
    }
  }
}
</script>
<style lang="scss" scoped>
@import '../dashboard/src/styles/variables';

.redtext {
  color: red;
}

.bluetext {
  color: $light-blue;
}

.info_title {
  font-weight: bold;
}

.card-h {
  height: 300px;
}
</style>
