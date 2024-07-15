<template>
  <div class="app-container">
    <div class="handle-search">
      <el-form :model="listQuery" inline @submit.native.prevent>
        <el-form-item label="名称">
          <el-input v-model="listQuery.name" placeholder="name" style="width: 150px;" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item :label="'path' | i18n">
          <el-input v-model="listQuery.path" placeholder="path" style="width: 150px;" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item label="存储库">
          <el-select v-model="listQuery.repositoryId" class="handle-select mr5" clearable placeholder="请选择">
            <el-option
              v-for="(item, index) in [{id: 0, name: '所有'}].concat(repositoryList)"
              :key="index"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="listQuery.status" class="handle-select mr5" clearable placeholder="请选择">
            <el-option
              v-for="(item, index) in [{status: 0, name: '所有'}].concat(status)"
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
    <div class="handle-box">
      <el-button type="primary" icon="el-icon-plus" class="mr5" @click="handleAdd">创建</el-button>
    </div>
    <el-table v-loading="listLoading" :data="list" border fit highlight-current-row style="width: 100%">
      <el-table-column align="center" :label="'ID' | i18n" width="80">
        <template slot-scope="scope">
          <span>{{ scope.row.id }}</span>
        </template>
      </el-table-column>

      <el-table-column prop="name" align="center" label="名称"/>

      <el-table-column prop="path" align="center" :label="'path' | i18n"/>

      <el-table-column prop="repositoryId" :formatter="filterRepo" align="center" :label="'repository' | i18n"/>

      <el-table-column prop="execTimeCron" align="center" label="执行时间">
        <template slot-scope="{row}">
          <span>{{ row.execTimeCron + '   ' }}</span>
          <el-button circle type="text" icon="el-icon-view" @click="cronNext(row.execTimeCron)"/>
        </template>
      </el-table-column>

      <el-table-column class-name="status-col" label="状态">
        <template slot-scope="{row}">
          <el-tag :type="row.status === 1 ? 'success' : 'warning'">
            {{ row.status === 1 ? '运行' : '停止' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" align="center" :label="'createdAt' | i18n" :formatter="dateFormat"/>
      <el-table-column align="center" label="操作" width="200">
        <template slot-scope="{row}">
          <el-button-group>
            <el-button type="success" size="small" @click="backupHandler(row.id)"
                       v-loading.fullscreen.lock="fullscreenLoading"
                       element-loading-text="正在执行,请勿关闭页面..."
                       element-loading-spinner="el-icon-loading">
              立即执行
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
        <el-form-item label="名称" prop="name">
          <el-input v-model="temp.name"/>
        </el-form-item>
        <el-form-item :label="'path' | i18n" prop="path">
          <el-input v-model="temp.path" disabled>
            <el-button slot="append" @click="openDirSelect()">选择</el-button>
          </el-input>
        </el-form-item>
        <el-form-item :label="'repository' | i18n" prop="repositoryId">
          <el-select v-model="temp.repositoryId" placeholder="请选择">
            <el-option v-for="item in repositoryList" :key="item.id" :label="item.name" :value="item.id"/>
          </el-select>
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-select v-model="temp.status" placeholder="请选择">
            <el-option v-for="item in status" :key="item.status" :label="item.name" :value="item.status"/>
          </el-select>
        </el-form-item>
        <el-form-item label="cron表达式" prop="execTimeCron">
          <el-popover v-model="cronPopover">
            <cron @change="changeCron" @close="cronPopover=false"/>
            <el-input
              slot="reference"
              v-model="temp.execTimeCron"
              placeholder="请输入定时策略"
              clearable
              @click="cronPopover=true"
            />
          </el-popover>
          <el-button type="text" @click="cronNext(temp.execTimeCron)">下次执行时间</el-button>
          <span style="margin-left: 20px;color: red">注意：最后一位"年"仅支持每年（*），其它值无效</span>
        </el-form-item>
        <el-form-item label="读取并发数量" prop="ReadConcurrency">
          <el-input v-model="temp.readConcurrency" clearable>
            <template slot="append">默认2</template>
          </el-input>
        </el-form-item>

        <el-form-item v-if="dialogStatus === 'create'" label="立即备份">
          <el-switch v-model="temp.immediate"/>
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogFormVisible = false">
          取消
        </el-button>
        <el-button
          type="primary"
          :loading="buttonLoading"
          @click=" dialogStatus === 'create' ? createData() : updateData()"
        >
          确定
        </el-button>
      </div>
    </el-dialog>
    <el-dialog
      title="下次执行时间"
      :visible.sync="dialogVisible"
      width="20%"
    >
      <div class="nexttime">
        <p v-for="item in dialogdata">{{ item }}</p>
      </div>

      <span slot="footer" class="dialog-footer">
        <el-button @click="dialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="dialogVisible = false">确 定</el-button>
      </span>
    </el-dialog>
    <el-dialog
      title="选择文件夹"
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
                    确定
                  </el-button>
          </span>
        </span>
        </div>
      </div>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogDirVisible = false">
          取消
        </el-button>
        <el-button
          type="primary"
          @click="confirmDirSelect()">
          确定
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
        {name: '运行', status: 1},
        {name: '停止', status: 2}
      ],
      dialogDirVisible: false,
      dirCur: "/data/",
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
        update: '修改',
        create: '创建'
      },
      cronPopover: false,
      dialogStatus: '',
      dialogFormVisible: false,
      buttonLoading: false,
      dialogVisible: false,
      dialogdata: '',
      temp: {
        name: '',
        path: '/data/',
        repositoryId: '',
        status: 2,
        immediate: false,
        execTimeCron: '',
        readConcurrency: 2
      },
      rules: {
        name: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        status: [{required: true, message: '请选择类型', trigger: 'change'}],
        path: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        execTimeCron: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        repositoryId: [{required: true, message: '该项为必填项', trigger: 'change'}]
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
        name: '根',
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
      let res = '存储库已删除'
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
          title: '备份中...',
          dangerouslyUseHTMLString: true,
          message: '请前往"<a style="color: #409EFF" href="/Task/index">任务记录</a>"查看'
        })
      }).finally(() => {
        this.fullscreenLoading = false
      })
    },
    handleDel(id) {
      this.$confirm('确认删除该计划吗？', '删除', {
        type: 'warning'
      }).then(() => {
        this.$notify.error("演示环境，不能执行操作")
      }).catch(() => {
        this.$notify.info('取消删除')
      })
    },
    updateData() {
      this.$refs['dataForm'].validate((valid) => {
        if (valid) {
          if (!this.temp.execTimeCron) {
            this.$notify.error('请输入定时备份cron表达式')
            return
          }
          this.$notify.error("演示环境，不能执行操作")
        }
      })
    },
    createData() {
      this.$refs['dataForm'].validate((valid) => {
        if (valid) {
          if (!this.temp.execTimeCron) {
            this.$notify.error('请输入定时备份cron表达式')
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
            this.$notify.success('创建成功！')
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
