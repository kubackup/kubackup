<template>
  <div class="app-container">
    <div class="handle-search">
      <el-form :model="listQuery" inline @submit.native.prevent>
        <el-form-item :label="'plan' | i18n">
          <el-select v-model="listQuery.planId" class="handle-select mr5" clearable placeholder="请选择">
            <el-option
              v-for="(item, index) in [{id: 0, name: '所有'}].concat(planList)"
              :key="index"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="listQuery.name" placeholder="name" style="width: 150px;" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="listQuery.status" class="handle-select mr5" clearable placeholder="请选择">
            <el-option
              v-for="(item, index) in [{status: -1, name: '所有'}].concat(statusList)"
              :key="index"
              :label="item.name"
              :value="item.status"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="el-icon-search" @click="handleFilter">查询</el-button>
        </el-form-item>
      </el-form>
    </div>
    <el-table :data="list" border fit highlight-current-row style="width: 100%">
      <el-table-column align="center" :label="'ID' | i18n" width="80">
        <template slot-scope="scope">
          <span>{{ scope.row.id }}</span>
        </template>
      </el-table-column>

      <el-table-column prop="name" align="center" label="名称"/>

      <el-table-column prop="createdAt" :formatter="dateFormat" align="center" :label="'createdAt' | i18n"/>

      <el-table-column prop="duration" align="center" :label="'duration' | i18n">
        <template slot-scope="{row}">
          <div>
            {{ (row.summary.totalDuration || '0:0') + '(' + (row.scanner.duration || '0:0') + ')' }}
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="dataAdd" align="center" label="数据">
        <template slot-scope="{row}">
          {{ row.summary.dataAdded || '0 B' }}
        </template>
      </el-table-column>
      <el-table-column prop="path" align="center" :label="'path' | i18n"/>

      <el-table-column prop="repositoryId" align="center" :formatter="filterRepo" :label="'repository' | i18n"/>

      <el-table-column prop="planId" align="center" :formatter="filterPlan" :label="'plan' | i18n"/>

      <el-table-column align="center" :label="'progress' | i18n">
        <template slot-scope="{row}">
          <el-progress
            type="dashboard"
            :width="50"
            :percentage="row.progress.percentDone?Number((row.progress.percentDone*100).toFixed(0)):0"
          />
        </template>
      </el-table-column>

      <el-table-column prop="status" align="center" :label="'status' | i18n">
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
               :title="'任务信息 '+(taskInfo.summary.snapshotId===undefined?'...':taskInfo.summary.snapshotId)"
               :visible.sync="dialogFormVisible" top="5vh"
               @close="closeSockjs"
               width="80%">
      <el-row :gutter="10">
        <el-col v-if="taskInfo.scanner && taskInfo.status!==3" :span="8">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title bluetext">扫描信息</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="24"><p>数据量：{{ taskInfo.scanner.dataSize }}</p></el-col>
              <el-col :span="24"><p>文件数量：{{ taskInfo.scanner.totalFiles }}</p></el-col>
              <el-col :span="24"><p class="redtext">耗时：{{ taskInfo.scanner.duration }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.status===3 && taskInfo.scannerError" :span="8">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title redtext">扫描错误</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="24"><p>错误信息：{{ taskInfo.scannerError.error }}</p></el-col>
              <el-col :span="24"><p>错误项：{{ taskInfo.scannerError.item }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.summary && taskInfo.status!==3" :span="16">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title bluetext">恢复信息</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="8"><p>
                数据量：{{ taskInfo.progress.bytesDone ? taskInfo.progress.bytesDone : taskInfo.scanner.dataSize }}</p>
              </el-col>
              <el-col :span="8"><p>文件数量：{{ taskInfo.progress.filesDone }}</p></el-col>
              <el-col v-if="taskInfo.progress.errorCount>0" :span="8" class="redtext"><p>
                错误数量：{{ taskInfo.progress.errorCount }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p class="redtext">耗时：{{ taskInfo.progress.secondsElapsed }}</p></el-col>
              <el-col v-if="taskInfo.progress.secondsRemaining" :span="8"><p class="redtext">
                剩余时间：{{ taskInfo.progress.secondsRemaining }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p>平均速度：{{ taskInfo.progress.avgSpeed }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.status===3 && (taskInfo.archivalError||taskInfo.restoreError)" :span="16">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title redtext">恢复错误</span>
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
               :title="'任务信息 '+(taskInfo.summary.snapshotId===undefined?'...':taskInfo.summary.snapshotId)"
               :visible.sync="dialogFormVisible" top="5vh"
               width="80%" @close="closeSockjs">
      <el-row :gutter="10">
        <el-col v-if="taskInfo.scanner && taskInfo.status!==3" :span="8">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title bluetext">扫描信息</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="24"><p>总数据量：{{ taskInfo.scanner.dataSize }}</p></el-col>
              <el-col :span="24"><p>本次变动文件数量：{{ taskInfo.scanner.totalFiles }}</p></el-col>
              <el-col :span="24"><p class="redtext">扫描耗时：{{ taskInfo.scanner.duration }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.status===3 && taskInfo.scannerError" :span="12">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title redtext">扫描错误</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="24"><p>错误信息：{{ taskInfo.scannerError.error }}</p></el-col>
              <el-col :span="24"><p>错误项：{{ taskInfo.scannerError.item }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.status!==3" :span="16">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title bluetext">汇总信息</span>
            </div>
            <el-row :gutter="10">
              <el-col :span="8"><p>
                总数据量：{{ taskInfo.progress.bytesDone ? taskInfo.progress.bytesDone : taskInfo.scanner.dataSize }}</p>
              </el-col>
              <el-col :span="8"><p>新增文件夹：{{ taskInfo.summary.dirsNew }}</p></el-col>
              <el-col :span="8"><p>新增文件：{{ taskInfo.summary.filesNew }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p class="redtext">新增数据量：{{ taskInfo.summary.dataAdded }}</p></el-col>
              <el-col :span="8"><p>变动文件夹：{{ taskInfo.summary.dirsChanged }}</p></el-col>
              <el-col :span="8"><p>变动文件：{{ taskInfo.summary.filesChanged }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p class="redtext">完成文件数量：{{ taskInfo.progress.filesDone }}</p></el-col>
              <el-col :span="8"><p>未修改文件夹：{{ taskInfo.summary.dirsUnmodified }}</p></el-col>
              <el-col :span="8"><p>未修改文件：{{ taskInfo.summary.filesUnmodified }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p class="redtext">耗时：{{ taskInfo.progress.secondsElapsed }}</p></el-col>
              <el-col v-if="taskInfo.progress.secondsRemaining && sockjsOpen" :span="8"><p class="redtext">
                剩余时间：{{ taskInfo.progress.secondsRemaining }}</p></el-col>
              <el-col v-if="taskInfo.progress.errorCount>0" :span="8" class="redtext"><p>
                错误数量：{{ taskInfo.progress.errorCount }}</p></el-col>
            </el-row>
            <el-row :gutter="10">
              <el-col :span="8"><p>平均速度：{{ taskInfo.progress.avgSpeed }}</p></el-col>
            </el-row>
          </el-card>
        </el-col>
        <el-col v-if="taskInfo.archivalError" :span="12">
          <el-card class="card-h">
            <div slot="header" class="clearfix">
              <span class="info_title redtext">备份错误</span>
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
              <span class="info_title">进度</span>
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
        {name: '新建', status: 0, color: 'primary'},
        {name: '运行中', status: 1, color: 'primary'},
        {name: '已完成', status: 2, color: 'success'},
        {name: '错误', status: 3, color: 'danger'}
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
              title: '错误',
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
