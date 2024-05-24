<template>
  <el-card style="height: 300px;">
    <div slot="header" class="clearfix">
      <span>个人中心</span>
    </div>

    <div class="user-profile">
      <div class="box-center">
        <pan-thumb :height="'100px'" :width="'100px'" :hoverable="false">
          <div>你好</div>
          {{ user.nickName }}
        </pan-thumb>
      </div>
      <div class="box-center">
        <div class="user-name text-center">{{ user.userName }}</div>
      </div>
      <div class="box-center">
        <el-button type="primary" @click="repwdHandler">修改密码</el-button>
      </div>
    </div>
    <el-dialog
      title="修改密码"
      :visible.sync="dialogFormVisible"
    >
      <el-form ref="dataForm" :rules="rules" :model="temp" label-position="left" label-width="120px">
        <el-form-item label="旧密码" prop="oldPassword">
          <el-input v-model="temp.oldPassword" type="password" clearable/>
        </el-form-item>
        <el-form-item label="新密码" prop="password">
          <el-input v-model="temp.password" type="password" clearable/>
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="temp.confirmPassword" type="password" clearable/>
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogFormVisible = false">
          取消
        </el-button>
        <el-button
          type="primary"
          @click="repwd"
        >
          确定
        </el-button>
      </div>
    </el-dialog>
  </el-card>
</template>

<script>
import PanThumb from '@/components/PanThumb'
import {fetchRePwd} from "@/api/user";

export default {
  components: {PanThumb},
  props: {
    user: {
      type: Object,
      default: () => {
        return {
          nickName: '',
          userName: '',
          email: '',
          phone: ''
        }
      }
    }
  },
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
      dialogFormVisible: false,
      temp: {
        oldPassword: "",
        password: "",
        confirmPassword: '',
      },
      rules: {
        oldPassword: [{required: true, message: '该项为必填项', trigger: 'blur'}],
        confirmPassword: [{required: true, validator: validatePassword, trigger: 'blur'}],
        password: [{required: true, message: '该项为必填项', trigger: 'blur'}],
      }
    }
  },
  methods: {
    resetTemp() {
      this.temp = {
        oldPassword: "",
        password: "",
        confirmPassword: '',
      }
    },
    repwdHandler() {
      this.resetTemp()
      this.dialogFormVisible = true
    },
    repwd() {
      this.$refs['dataForm'].validate((valid) => {
        if (valid) {
          fetchRePwd(this.temp).then(res => {
            this.$notify.success({
              title: '提示',
              message: res.data
            })
          }).finally(() => {
            this.dialogFormVisible = false
          })
        }
      })
    }
  }
}
</script>

<style lang="scss" scoped>
.box-center {
  margin: 0 auto;
  display: table;
}

.text-muted {
  color: #777;
}

.user-profile {
  .user-name {
    font-weight: bold;
  }

  .box-center {
    padding-top: 10px;
  }

  .user-role {
    padding-top: 10px;
    font-weight: 400;
    font-size: 14px;
  }

  .box-social {
    padding-top: 30px;

    .el-table {
      border-top: 1px solid #dfe6ec;
    }
  }

  .user-follow {
    padding-top: 20px;
  }
}

.user-bio {
  margin-top: 20px;
  color: #606266;

  span {
    padding-left: 4px;
  }

  .user-bio-section {
    font-size: 14px;
    padding: 15px 0;

    .user-bio-section-header {
      border-bottom: 1px solid #dfe6ec;
      padding-bottom: 10px;
      margin-bottom: 10px;
      font-weight: bold;
    }
  }
}
</style>
