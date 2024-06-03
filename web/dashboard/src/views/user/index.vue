<template>
  <div class="app-container">
    <div class="handle-box">
      <el-button type="success" icon="el-icon-plus" class="mr5" @click="handleAdd">创建</el-button>
      <el-button type="primary" icon="el-icon-search" @click="getList">查询</el-button>
    </div>
    <el-table v-loading="listLoading" :data="list" border fit highlight-current-row style="width: 100%">
      <el-table-column prop="id" align="center" label="ID"/>
      <el-table-column prop="userName" align="center" label="账号"/>
      <el-table-column prop="nickName" align="center" label="用户名"/>
      <el-table-column prop="email" align="center" label="邮箱" width="220"/>
      <el-table-column prop="phone" align="center" label="手机号码"/>
      <el-table-column prop="lastLogin" :formatter="dateFormat" align="center" label="最后登录时间"/>

      <el-table-column align="center" label="操作" width="200">
        <template slot-scope="{row}">
          <el-button-group>
            <el-button type="primary" size="small" icon="el-icon-edit" class="mr5"
                       @click="handleEdit(row)">
              修改
            </el-button>
            <el-button type="danger" size="small" icon="el-icon-delete" @click="handleDel(row.id)">
              删除
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
        <el-form-item label="账号" prop="userName">
          <el-input v-model="temp.userName" :disabled="dialogStatus === 'update'"/>
        </el-form-item>
        <el-form-item label="用户名" prop="nickName">
          <el-input v-model="temp.nickName"/>
        </el-form-item>
        <el-form-item label="密码" prop="password" v-if="dialogStatus === 'create'">
          <el-input v-model="temp.password" type="password"/>
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword" v-if="dialogStatus === 'create'">
          <el-input v-model="temp.confirmPassword" type="password"/>
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="temp.email" type="email"/>
        </el-form-item>
        <el-form-item label="手机号码" prop="phone">
          <el-input v-model="temp.phone"/>
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
import {fetchCreate, fetchDel, fetchList, fetchUpdate} from '@/api/user'
import {dateFormat} from "@/utils";

export default {
  name: 'UserList',
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
      listLoading: false,
      list: [],
      textMap: {
        update: '修改',
        create: '创建'
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
        userName: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        nickName: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        confirmPassword: [{required: true,validator: validatePassword, trigger: 'blur'}],
        password: [{required: true,message: '该项为必填项', trigger: 'blur'}],
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
          fetchUpdate(this.temp).then(() => {
            this.$notify.success('修改成功！')
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
      this.$confirm('确认删除该用户吗？', '删除', {
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
