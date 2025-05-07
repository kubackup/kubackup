<template>
  <div class="app-container">
    <div class="handle-search">
      <el-form :model="listQuery" inline @submit.native.prevent>
        <el-form-item :label="$t('msg.name')">
          <el-input v-model="listQuery.name" :placeholder="$t('msg.name')" style="width: 150px;" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item :label="$t('msg.path')">
          <el-input v-model="listQuery.path" :placeholder="$t('msg.path')" style="width: 150px;" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item :label="$t('msg.title.repository')">
          <el-select v-model="listQuery.repositoryId" class="handle-select mr5" clearable :placeholder="$t('msg.pleaseSelect')">
            <el-option
              v-for="(item, index) in [{id: 0, name: $t('msg.all')}].concat(repositoryList)"
              :key="index"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('msg.statusLabel')">
          <el-select v-model="listQuery.status" class="handle-select mr5" clearable :placeholder="$t('msg.pleaseSelect')">
            <el-option
              v-for="(item, index) in [{status: 0, name: $t('msg.all')}].concat(status)"
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
    <div class="handle-box">
      <el-button type="primary" icon="el-icon-plus" class="mr5" @click="handleAdd">{{ $t('msg.createAction') }}</el-button>
    </div>
    <el-table v-loading="listLoading" :data="list" border fit highlight-current-row style="width: 100%">
      <el-table-column align="center" :label="$t('msg.id')" width="80">
        <template slot-scope="scope">
          <span>{{ scope.row.id }}</span>
        </template>
      </el-table-column>

      <el-table-column prop="name" align="center" :label="$t('msg.name')"/>

      <el-table-column prop="path" align="center" :label="$t('msg.path')"/>

      <el-table-column prop="repositoryId" :formatter="filterRepo" align="center" :label="$t('msg.repositoryId')"/>

      <el-table-column prop="execTimeCron" align="center" :label="$t('msg.execTimeCron')">
        <template slot-scope="{row}">
          <span>{{ row.execTimeCron + '   ' }}</span>
          <el-button circle type="text" icon="el-icon-view" @click="cronNext(row.execTimeCron)"/>
        </template>
      </el-table-column>

      <el-table-column class-name="status-col" :label="$t('msg.statusLabel')">
        <template slot-scope="{row}">
          <el-tag :type="row.status === 1 ? 'success' : 'warning'">
            {{ row.status === 1 ? $t('msg.run') : $t('msg.stop') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" align="center" :label="$t('msg.createdAt')" :formatter="dateFormat"/>
      <el-table-column align="center" :label="$t('msg.title.operationAction')" width="200">
        <template slot-scope="{row}">
          <el-button-group>
            <el-button type="success" size="small" @click="backupHandler(row.id)"
                       v-loading.fullscreen.lock="fullscreenLoading"
                       :element-loading-text="$t('msg.tips.runingMsg')"
                       element-loading-spinner="el-icon-loading">
              {{ $t('msg.runNow')}}
            </el-button>
            <el-button type="primary" size="small" icon="el-icon-edit" class="mr5" @click="handleEdit(row)"/>
            <el-button type="danger" size="small" icon="el-icon-delete" @click="handleDel(row.id)"/>
          </el-button-group>
        </template>
      </el-table-column>
    </el-table>
    <pagination
      v-show="total > 0"
      :total="total"
      :page.sync="listQuery.pageNum"
      :limit.sync="listQuery.pageSize"
      @pagination="getList"
    />

    <el-dialog
      v-el-drag-dialog
      :title="textMap[dialogStatus]"
      :visible.sync="dialogFormVisible"
      @dragDialog="handleDrag"
    >
      <el-form ref="dataForm" :rules="rules" :model="temp" label-position="left" label-width="120px">
        <el-form-item :label="$t('msg.name')" prop="name">
          <el-input v-model="temp.name"/>
        </el-form-item>
        <el-form-item :label="$t('msg.path')" prop="path">
          <el-input v-model="temp.path" disabled>
            <el-button slot="append" @click="openDirSelect()">{{ $t('msg.select') }}</el-button>
          </el-input>
        </el-form-item>
        <el-form-item :label="$t('msg.title.repository')" prop="repositoryId">
          <el-select v-model="temp.repositoryId" :placeholder="$t('msg.pleaseSelect')">
            <el-option v-for="item in repositoryList" :key="item.id" :label="item.name" :value="item.id"/>
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('msg.statusLabel')" prop="status">
          <el-select v-model="temp.status" :placeholder="$t('msg.pleaseSelect')">
            <el-option v-for="item in status" :key="item.status" :label="item.name" :value="item.status"/>
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('msg.cron')" prop="execTimeCron">
          <el-popover v-model="cronPopover">
            <cron @change="changeCron" @close="cronPopover=false"/>
            <el-input
              slot="reference"
              v-model="temp.execTimeCron"
              :placeholder="$t('msg.pleaseInput')+$t('msg.cron')"
              clearable
              @click="cronPopover=true"
            />
          </el-popover>
          <el-button type="text" @click="cronNext(temp.execTimeCron)">{{ $t('msg.nextTriggerTime') }}</el-button>
          <span style="margin-left: 20px;color: red">{{ $t('msg.tips.cronNotice') }}</span>
        </el-form-item>
        <el-form-item :label="$t('msg.readConcurrency')" prop="readConcurrency">
          <el-input v-model="temp.readConcurrency" clearable>
            <template slot="append">{{ $t('msg.default') }} 2</template>
          </el-input>
        </el-form-item>

        <el-form-item v-if="dialogStatus === 'create'" :label="$t('msg.backupNow')">
          <el-switch v-model="temp.immediate"/>
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogFormVisible = false">
          {{ $t('msg.cancel') }}
        </el-button>
        <el-button
          type="primary"
          :loading="buttonLoading"
          @click=" dialogStatus === 'create' ? createData() : updateData()"
        >
          {{ $t('msg.confirm') }}
        </el-button>
      </div>
    </el-dialog>
    <el-dialog
      :title="$t('msg.nextTriggerTime')"
      :visible.sync="dialogVisible"
      width="20%"
    >
      <div class="nexttime">
        <p v-for="item in dialogdata">{{ item }}</p>
      </div>

      <span slot="footer" class="dialog-footer">
        <el-button @click="dialogVisible = false">{{ $t('msg.cancel') }}</el-button>
        <el-button type="primary" @click="dialogVisible = false">{{ $t('msg.confirm') }}</el-button>
      </span>
    </el-dialog>
    <el-dialog
      :title="$t('msg.selectDir')"
      :visible.sync="dialogDirVisible"
    >
      <div>
        <el-breadcrumb separator="/">
          <el-breadcrumb-item class="breadcrumb-item" v-for="(item, index) in getDirSpea()" :key="index">
            <span class="title" @click="lsDir(item.path,true)">{{ item.name }}</span>
          </el-breadcrumb-item>
        </el-breadcrumb>
        <div class="filenodes">
          <span class="custom-tree-node filenode" v-for="(item, index) in dirList" :key="index" v-if="item.isDir"
                :class="{active : dirCur === item.path}"
                @dblclick.prevent="lsDir(item.path,item.isDir)"
                @click="selectDir(item.path,item.isDir)">
          <div class="file-title">
            <i v-if="item.isDir" class="el-icon-folder"/>
            <span style="margin-left: 5px;user-select: none;">{{ item.name }}</span>
          </div>
          <span>
            <el-button
              type="text"
              class="confirmbtn"
              size="mini"
              @click="confirmDirSelect(item.path)">
                    {{ $t('msg.confirm') }}
                  </el-button>
          </span>
        </span>
        </div>
      </div>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogDirVisible = false">
          {{ $t('msg.cancel') }}
        </el-button>
        <el-button
          type="primary"
          @click="confirmDirSelect()">
          {{ $t('msg.confirm') }}
        </el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import {fetchCreate, fetchDel, fetchList, fetchNextTime, fetchUpdate} from '@/api/plan'
import {fetchList as repolist} from '@/api/repository'
import Pagination from '@/components/Pagination'
import {dateFormat} from '@/utils'
import elDragDialog from '@/directive/el-drag-dialog'
import {cron} from 'vue-cron'
import {fetchBackup} from '@/api/task'
import {fetchLs} from "@/api/system";

export default {
  name: 'Plan',
  directives: {elDragDialog},
  components: {Pagination, cron},
  data() {
    return {
      status: [
        {name: this.$t('msg.run'), status: 1},
        {name: this.$t('msg.stop'), status: 2}
      ],
      dialogDirVisible: false,
      dirCur: "/",
      dirList: [],
      fullscreenLoading: false,
      repositoryList: [],
      list: [],
      total: 0,
      listLoading: false,
      listQuery: {
        name: '',
        type: '',
        repositoryId: 0,
        status: 0,
        pageNum: 1,
        pageSize: 10
      },
      textMap: {
        update: this.$t('msg.operation.update'),
        create: this.$t('msg.operation.create')
      },
      cronPopover: false,
      dialogStatus: '',
      dialogFormVisible: false,
      buttonLoading: false,
      dialogVisible: false,
      dialogdata: '',
      temp: {
        name: '',
        path: '/',
        repositoryId: '',
        status: 2,
        immediate: false,
        execTimeCron: '',
        readConcurrency: 2
      },
      rules: {
        name: [{required: true, message: this.$t('msg.tips.emptyError'), trigger: 'blur'}],
        status: [{required: true, message: this.$t('msg.pleaseSelectType'), trigger: 'change'}],
        path: [{required: true, message: this.$t('msg.tips.emptyError'), trigger: 'blur'}],
        execTimeCron: [{required: true, message: this.$t('msg.tips.emptyError'), trigger: 'blur'}],
        repositoryId: [{required: true, message: this.$t('msg.tips.emptyError'), trigger: 'change'}]
      }
    }
  },
  created() {
    this.getList()
    repolist().then(response => {
      this.repositoryList = response.data
    })
  },
  methods: {
    dateFormat(row, column, cellValue, index) {
      return dateFormat(cellValue, 'yyyy-MM-dd hh:mm')
    },
    handleDrag() {
      this.$refs.select.blur()
    },
    changeCron(val) {
      this.temp.execTimeCron = val
    },
    openDirSelect() {
      this.dialogDirVisible = true
      this.dirCur = this.temp.path
      this.dirList = []
      this.lsDir(this.dirCur, true)
    },
    confirmDirSelect(path) {
      if (path) {
        this.dirCur = path
      }
      this.temp.path = this.dirCur
      if (!this.temp.name) {
        this.temp.name = this.dirCur
      }
      this.dialogDirVisible = false
    },
    getDirSpea() {
      var dirs = this.dirCur.split("/")
      dirs.shift()
      var res = []
      var path = ''
      dirs.forEach(n => {
        if (n === '') {
          return
        }
        path = path + '/' + n
        res.push({
          name: n,
          path: path
        })
      })
      res.unshift({
        name: this.$t('msg.root'),
        path: '/'
      })
      return res
    },
    selectDir(path, isdir) {
      if (!isdir) {
        return
      }
      this.dirCur = path
    },
    lsDir(path, isdir) {
      if (!isdir) {
        return
      }
      this.dirCur = path
      var q = {
        path: this.dirCur
      }
      fetchLs(q).then(res => {
        this.dirList = res.data
      })
    },
    cronNext(str) {
      this.dialogdata = []
      var q = {
        cron: str
      }
      fetchNextTime(q).then(res => {
        this.dialogVisible = true
        this.dialogdata = res.data
      })
    },
    filterRepo(row, column, cellValue, index) {
      let res = this.$t('msg.tips.repositoryNotFound')
      this.repositoryList.forEach(value => {
        if (value.id === cellValue) {
          res = value.name
          return res
        }
      })
      return res
    },
    getList() {
      this.listLoading = true
      fetchList(this.listQuery).then(response => {
        this.list = response.data.items
        this.total = response.data.total
      }).finally(() => {
        this.listLoading = false
      })
    },
    handleFilter() {
      this.listQuery.pageNum = 1
      this.getList()
    },
    resetTemp() {
      this.temp = {
        name: '',
        path: '/',
        repositoryId: '',
        status: 2,
        immediate: false,
        execTimeCron: ''
      }
    },
    handleEdit(row) {
      this.temp = Object.assign({}, row)
      this.dialogStatus = 'update'
      this.dialogFormVisible = true
      this.$nextTick(() => {
        this.$refs['dataForm'].clearValidate()
      })
    },
    handleAdd() {
      this.resetTemp()
      this.dialogStatus = 'create'
      this.dialogFormVisible = true
      this.$nextTick(() => {
        this.$refs['dataForm'].clearValidate()
      })
    },
    backupHandler(planid) {
      this.fullscreenLoading = true
      fetchBackup(planid).then(() => {
        this.$notify.success({
          title: this.$t('msg.tips.backuping'),
          dangerouslyUseHTMLString: true,
          message: '<a style="color: #409EFF" href="/Task/index">'+this.$t('msg.tasks')+'</a>'
        })
      }).finally(() => {
        this.fullscreenLoading = false
      })
    },
    handleDel(id) {
      this.$confirm(this.$t('msg.tips.confirmDel')+this.$t('msg.plan')+'？', this.$t('msg.operation.delete'), {
        type: 'warning'
      }).then(() => {
        this.listLoading = true
        fetchDel(id).then(() => {
          this.$notify.success(this.$t('msg.success'))
          this.getList()
        }).finally(() => {
          this.listLoading = false
        })
      }).catch(() => {
        this.$notify.info(this.$t('msg.cancel'))
      })
    },
    updateData() {
      this.$refs['dataForm'].validate((valid) => {
        if (valid) {
          if (!this.temp.execTimeCron) {
            this.$notify.error(this.$t('msg.tips.cronErr'))
            return
          }
          this.buttonLoading = true
          if (this.temp.readConcurrency === '') {
            this.temp.readConcurrency = 2
          } else {
            this.temp.readConcurrency = Number(this.temp.readConcurrency)
          }
          fetchUpdate(this.temp).then(() => {
            this.$notify.success(this.$t('msg.success'))
            this.buttonLoading = false
            this.dialogFormVisible = false
            this.getList()
          }).catch(() => {
            this.buttonLoading = false
          })
        }
      })
    },
    createData() {
      this.$refs['dataForm'].validate((valid) => {
        if (valid) {
          if (!this.temp.execTimeCron) {
            this.$notify.error(this.$t('msg.tips.cronErr'))
            return
          }
          this.buttonLoading = true
          if (this.temp.readConcurrency === '') {
            this.temp.readConcurrency = 2
          } else {
            this.temp.readConcurrency = Number(this.temp.readConcurrency)
          }
          fetchCreate(this.temp).then(res => {
            var planid = res.data
            this.$notify.success(this.$t('msg.success'))
            this.getList()
            if (this.temp.immediate) {
              this.backupHandler(planid)
            }
          }).finally(() => {
            this.buttonLoading = false
            this.dialogFormVisible = false
          })
        }
      })
    }
  }
}
</script>

<style lang="scss" scoped>
@import "src/styles/variables";

.custom-tree-node {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 5px;

  .file-title {
    font-size: 14px;
    padding-right: 8px;
  }
}

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

.filenode:hover {
  background-color: $light-blue;
  cursor: pointer;
  color: $menuHover;
}

.breadcrumb-item {
  padding: 5px;
}

.breadcrumb-item:hover {
  background-color: $light-blue;
  cursor: pointer;
  color: $menuHover;

  .title {
    color: $menuHover;
  }
}

.nexttime {
  text-align: center;
}

.bottom .value {
  margin-right: 20px !important;
}
</style>
