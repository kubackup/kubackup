<template>
  <div class="app-container">
    <div class="handle-box">
      <el-button type="success" icon="el-icon-plus" class="mr5" @click="handleAdd">{{ $t('msg.createAction') }}</el-button>
    </div>
    <el-table v-loading="listLoading" :data="list" border fit highlight-current-row style="width: 100%">
      <el-table-column prop="id" align="center" :label="$t('msg.id')"/>
      <el-table-column prop="userName" align="center" :label="$t('msg.login.username')"/>
      <el-table-column prop="nickName" align="center" :label="$t('msg.title.user')"/>
      <el-table-column prop="email" align="center" :label="$t('msg.email')" width="220"/>
      <el-table-column prop="phone" align="center" :label="$t('msg.phone')"/>
      <el-table-column prop="lastLogin" :formatter="dateFormat" align="center" :label="$t('msg.lastLogin')"/>

      <el-table-column align="center" :label="$t('msg.title.operationAction')" width="200">
        <template slot-scope="{row}">
          <el-button-group>
            <el-button type="primary" size="small" icon="el-icon-edit" class="mr5"
                       @click="handleEdit(row)">
              {{ $t('msg.operation.edit') }}
            </el-button>
            <el-button type="danger" size="small" icon="el-icon-delete" @click="handleDel(row.id)">
              {{ $t('msg.operation.delete') }}
            </el-button>
          </el-button-group>


        </template>
      </el-table-column>
    </el-table>

    <el-dialog :title="textMap[dialogStatus]" :visible.sync="dialogFormVisible">
      <el-form
        ref="dataForm"
        :rules="rules"
        :model="temp"
        label-position="left"
        label-width="90px"
        style="width: 400px; margin-left:50px;"
      >
        <el-form-item :label="$t('msg.login.username')" prop="userName">
          <el-input v-model="temp.userName" :disabled="dialogStatus === 'update'"/>
        </el-form-item>
        <el-form-item :label="$t('msg.title.user')" prop="nickName">
          <el-input v-model="temp.nickName"/>
        </el-form-item>
        <el-form-item :label="$t('msg.login.password')" prop="password" v-if="dialogStatus === 'create'">
          <el-input v-model="temp.password" type="password"/>
        </el-form-item>
        <el-form-item :label="$t('msg.reposPwdSec')" prop="confirmPassword" v-if="dialogStatus === 'create'">
          <el-input v-model="temp.confirmPassword" type="password"/>
        </el-form-item>
        <el-form-item :label="$t('msg.email')" prop="email">
          <el-input v-model="temp.email" type="email"/>
        </el-form-item>
        <el-form-item :label="$t('msg.phone')" prop="phone">
          <el-input v-model="temp.phone"/>
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
  </div>
</template>

<script>
import {fetchCreate, fetchDel, fetchList, fetchUpdate} from '@/api/user'
import {dateFormat} from "@/utils";

export default {
  name: 'UserList',
  data() {
    const validatePassword = (rule, value, callback) => {
      if (value === '') {
        callback(new Error(this.$t('msg.tips.emptyError')))
      } else if (value !== this.temp.password) {
        callback(new Error(this.$t('msg.tips.pwdSecError')))
      } else {
        callback()
      }
    }
    return {
      listLoading: false,
      list: [],
      textMap: {
        update: this.$t('msg.operation.update'),
        create: this.$t('msg.operation.create')
      },
      dialogStatus: '',
      dialogFormVisible: false,
      buttonLoading: false,
      temp: {
        userName: "",
        nickName: "",
        password: "",
        confirmPassword:'',
        email: "",
        phone: "",
      },
      rules: {
        userName: [{required: true, message: this.$t('msg.tips.emptyError'), trigger: 'blur'}],
        nickName: [{required: true, message: this.$t('msg.tips.emptyError'), trigger: 'blur'}],
        confirmPassword: [{required: true,validator: validatePassword, trigger: 'blur'}],
        password: [{required: true,message: this.$t('msg.tips.emptyError'), trigger: 'blur'}],
      }
    }
  },
  created() {
    this.getList()
  },
  methods: {
    resetTemp() {
      this.temp = {
        userName: "",
        nickName: "",
        password: "",
        confirmPassword:'',
        email: "",
        phone: "",
      }
    },
    dateFormat(row, column, cellValue, index) {
      return dateFormat(cellValue, 'yyyy-MM-dd hh:mm')
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
          fetchCreate(this.temp).then(() => {
            this.$notify.success(this.$t('msg.success') + '！')
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
          fetchUpdate(this.temp).then(() => {
            this.$notify.success(this.$t('msg.success') + '！')
            this.buttonLoading = false
            this.dialogFormVisible = false
            this.getList()
          }).catch(() => {
            this.buttonLoading = false
            this.dialogFormVisible = false
          })
        }
      })
    },
    handleDel(id) {
      this.$confirm(this.$t('msg.tips.confirmDel') + this.$t('msg.title.user') + '？', this.$t('msg.operation.delete'), {
        type: 'warning'
      }).then(() => {
        this.listLoading = true
        fetchDel(id).then(() => {
          this.$notify.success(this.$t('msg.success') + '！')
          this.getList()
        }).finally(() => {
          this.listLoading = false
        })
      }).catch(() => {
        this.$notify.info(this.$t('msg.cancel'))
      })
    },

    getList() {
      this.listLoading = true
      fetchList().then(response => {
        this.list = response.data
      }).finally(() => {
        this.listLoading = false
      })
    }
  }
}
</script>

<style scoped>

</style>
