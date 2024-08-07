<template>
  <div class="app-container">
    <div class="handle-search">
      <el-form :model="listQuery" inline @submit.native.prevent>
        <el-form-item label="名称">
          <el-input v-model="listQuery.name" placeholder="name" style="width: 150px;" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item :label="'type' | i18n">
          <el-select v-model="listQuery.type" class="handle-select mr5" placeholder="请选择">
            <el-option
              v-for="(item, index) in [{code: '', name: '所有'}].concat(typeList)"
              :key="index"
              :label="item.name"
              :value="item.code"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="el-icon-search" @click="getList">查询</el-button>
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

      <el-table-column prop="createdAt" align="center" :formatter="dateFormat" label="创建时间"/>
      <el-table-column prop="endPoint" align="left" label="服务器"/>
      <el-table-column class-name="status-col" label="存储类型" width="110">
        <template slot-scope="{row}">
          {{ formatType(row.type).name }}
        </template>
      </el-table-column>
      <el-table-column class-name="status-col" label="压缩模式" width="110">
        <template slot-scope="{row}">
          <el-tag :type="formatCompression(row.compression).color">
            {{ formatCompression(row.compression).name }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column class-name="status-col" label="连接状态" width="110">
        <template slot-scope="{row}">
          <el-tooltip class="item" v-if="row.errmsg" effect="dark" :content="row.errmsg" placement="bottom">
            <el-tag :type="formatStatus(row.status).color">
              {{ formatStatus(row.status).name }}
            </el-tag>
          </el-tooltip>
          <el-tag v-else :type="formatStatus(row.status).color">
            {{ formatStatus(row.status).name }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column align="center" label="操作">
        <template slot-scope="{row}">
          <el-dropdown trigger="click" hide-on-click @command="handleCmd">
            <span class="el-dropdown-link">
              操作<i class="el-icon-arrow-down el-icon--right"></i>
            </span>
            <el-dropdown-menu slot="dropdown">
              <el-dropdown-item icon="el-icon-video-camera" :command="{cmd:'restore',data:row.id}">恢复
              </el-dropdown-item>
              <el-dropdown-item icon="el-icon-setting" :command="{cmd:'oper',data:row.id}">维护
              </el-dropdown-item>
              <el-dropdown-item icon="el-icon-video-camera" :command="{cmd:'snap',data:row.id}">快照
              </el-dropdown-item>
              <el-dropdown-item icon="el-icon-video-camera" :command="{cmd:'edit',data:row}">编辑
              </el-dropdown-item>
              <el-dropdown-item icon="el-icon-delete" :command="{cmd:'del',data:row.id}">删除
              </el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog :title="textMap[dialogStatus]" :visible.sync="dialogFormVisible" top="5vh">
      <el-form ref="dataForm" :rules="rules" :model="temp" label-position="left" label-width="220px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="temp.name" clearable/>
        </el-form-item>
        <el-form-item label="存储类型" prop="type">
          <el-select v-model="temp.type" placeholder="请选择" @change="this.onTypeChange">
            <el-option v-for="item in typeList" :key="item.code" :label="item.name" :value="item.code"/>
          </el-select>
          <span class="repo-type-tips">{{ formatType(temp.type).tips }}</span>
        </el-form-item>
        <el-form-item label="端点（endPoint）" prop="endPoint">
          <el-input v-model="temp.endPoint" :placeholder="endPointPlaceholder" clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===1||temp.type===2" label="地区（region）" prop="region">
          <el-input v-model="temp.region" clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===1||temp.type===2||temp.type===6" label="空间（bucket）" prop="bucket">
          <el-input v-model="temp.bucket" clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===6" label="Access Key" prop="keyId">
          <el-input v-model="temp.keyId" clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===6" label="Secret Key" prop="secret">
          <el-input v-model="temp.secret" type="password" show-password clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===1||temp.type===2" label="AWS_ACCESS_KEY_ID" prop="keyId">
          <el-input v-model="temp.keyId" clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===1||temp.type===2" label="AWS_SECRET_ACCESS_KEY" prop="secret">
          <el-input v-model="temp.secret" type="password" show-password clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===7" label="SecretID" prop="keyId">
          <el-input v-model="temp.keyId" clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===7" label="SecretKey" prop="secret">
          <el-input v-model="temp.secret" type="password" show-password clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===5" label="账号" prop="keyId">
          <el-input v-model="temp.keyId" clearable/>
        </el-form-item>
        <el-form-item v-if="temp.type===5" label="密码" prop="secret">
          <el-input v-model="temp.secret" type="password" show-password clearable/>
        </el-form-item>
        <el-form-item v-if="dialogStatus === 'create'" label="仓库密码" prop="password">
          <el-input v-model="temp.password" show-password clearable type="password"/>
        </el-form-item>
        <el-form-item v-if="dialogStatus === 'create'" label="确认密码" prop="confirmPassword">
          <el-input v-model="temp.confirmPassword" show-password clearable type="password"/>
        </el-form-item>
        <el-form-item v-if="dialogStatus === 'create'" label="压缩模式" prop="type">
          <el-select v-model="temp.compression" placeholder="请选择">
            <el-option v-for="item in compressionList" :key="item.code" :label="item.name" :value="item.code"/>
          </el-select>
        </el-form-item>
        <el-form-item label="PackSize" prop="PackSize">
          <el-input v-model="temp.packSize" clearable>
            <template slot="append">MiB</template>
          </el-input>
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
  </div>
</template>

<script>
import {fetchCreate, fetchDel, fetchList, fetchUpdate} from '@/api/repository'
import {dateFormat} from '@/utils'
import {repoStatusList, repoTypeList, compressionList} from "@/consts";

export default {
  name: 'RepositoryList',
  data() {
    const validatePassword = (rule, value, callback) => {
      if (value === '') {
        callback(new Error('该项为必填项'))
      } else if (value !== this.temp.password) {
        callback(new Error('两次密码输入不一致'))
      } else {
        callback()
      }
    }
    return {
      typeList: repoTypeList,
      statusList: repoStatusList,
      compressionList: compressionList,
      list: [],
      listLoading: false,
      listQuery: {
        name: '',
        type: '',
        pageNum: 1,
        pageSize: 10
      },
      textMap: {
        update: '修改存储库',
        create: '创建存储库'
      },
      dialogStatus: '',
      dialogFormVisible: false,
      buttonLoading: false,
      endPointPlaceholder: '',
      temp: {
        id: 0,
        name: '',
        type: 4,
        endPoint: '',
        region: '',
        bucket: '',
        keyId: '',
        secret: '',
        projectId: '',
        accountName: '',
        accountKey: '',
        accountId: '',
        password: '',
        confirmPassword: '',
        compression: 0,
        packSize: 16
      },
      rules: {
        name: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        type: [{required: true, message: '请选择类型', trigger: 'change'}],
        endPoint: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        region: [{required: false, message: '该项为必填项', trigger: 'blur'}],
        bucket: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        keyId: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        secret: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        password: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        confirmPassword: [{required: true, validator: validatePassword, trigger: 'blur'}],
      }
    }
  },
  created() {
    this.getList()
  },
  methods: {
    dateFormat(row, column, cellValue, index) {
      return dateFormat(cellValue, 'yyyy-MM-dd hh:mm')
    },
    resetTemp() {
      this.temp = {
        id: 0,
        name: '',
        type: 4,
        endPoint: '',
        region: '',
        bucket: '',
        keyId: '',
        secret: '',
        projectId: '',
        accountName: '',
        accountKey: '',
        accountId: '',
        password: '',
        user: '',
        authpwd: '',
        compression: 0,
        packSize: 16
      }
      this.endPointPlaceholder = ''
    },
    onTypeChange(val) {
      this.endPointPlaceholder = ''
      switch (val) {
        case 1:
          this.endPointPlaceholder = 'http(s)://s3host:port'
          break
        case 2:
          this.endPointPlaceholder = 'https://<OSS-ENDPOINT>'
          break
        case 3:
          this.endPointPlaceholder = 'user@host:/data/my_backup_repo'
          break
        case 4:
          this.endPointPlaceholder = '/data/my_backup_repo'
          break
        case 5:
          this.endPointPlaceholder = 'http(s)://host:8000/my_backup_repo/'
          break
      }
    },
    handleCmd(datas) {
      const cmd = datas.cmd
      const data = datas.data
      switch (cmd) {
        case 'snap':
          this.$router.push('/repository/snapshot/' + data)
          break
        case 'restore':
          this.$router.push('/repository/restore/' + data)
          break
        case 'del':
          this.handleDel(data)
          break
        case 'edit':
          this.handleEdit(data)
          break
        case 'oper':
          this.$router.push('/repository/operation/' + data)
          break
      }
    },
    handleAdd() {
      this.resetTemp()
      this.dialogStatus = 'create'
      this.dialogFormVisible = true
      this.$nextTick(() => {
        this.$refs['dataForm'].clearValidate()
      })
    },
    createData() {
      this.$refs['dataForm'].validate((valid) => {
        if (valid) {
          this.buttonLoading = true
          this.temp.packSize = Number(this.temp.packSize)
          fetchCreate(this.temp).then(() => {
            this.$notify.success('创建成功！')
            this.buttonLoading = false
            this.dialogFormVisible = false
            this.getList()
          }).catch(() => {
            this.buttonLoading = false
          })
        }
      })
    },
    handleEdit(row) {
      this.temp = Object.assign({}, row)
      this.dialogStatus = 'update'
      this.dialogFormVisible = true
      this.$nextTick(() => {
        this.$refs['dataForm'].clearValidate()
      })
    },
    updateData() {
      this.$refs['dataForm'].validate((valid) => {
        if (valid) {
          this.buttonLoading = true
          this.temp.packSize = Number(this.temp.packSize)
          fetchUpdate(this.temp).then(() => {
            this.$notify.success('修改成功！')
            this.buttonLoading = false
            this.dialogFormVisible = false
            this.getList()
          }).catch(() => {
            this.buttonLoading = false
          })
        }
      })
    },
    handleDel(id) {
      this.$confirm('确认删除该存储库吗？', '删除', {
        type: 'warning'
      }).then(() => {
        this.listLoading = true
        fetchDel(id).then(() => {
          this.$notify.success('删除成功！')
          this.getList()
        }).finally(() => {
          this.listLoading = false
        })
      }).catch(() => {
        this.$notify.info('取消删除')
      })
    },
    formatType(code) {
      return this.typeList.find(item => item.code === code)
    },
    formatCompression(code) {
      return this.compressionList.find(item => item.code === code)
    },
    formatStatus(code) {
      return this.statusList.find(item => item.code === code)
    },
    getList() {
      this.listLoading = true
      fetchList(this.listQuery).then(response => {
        this.list = response.data
      }).finally(() => {
        this.listLoading = false
      })
    }
  }
}
</script>

<style scoped>
.el-dropdown-link {
  cursor: pointer;
  color: #409EFF;
}

.el-icon-arrow-down {
  font-size: 12px;
}

.repo-type-tips {
  margin-left: 10px;
  color: red;
}

</style>
