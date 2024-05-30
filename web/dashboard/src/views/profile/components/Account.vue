<template>
  <el-card style="padding: 20px">
    <el-form>
      <el-form-item label="账号">
        <el-input v-model.trim="tmp.userName"/>
      </el-form-item>
      <el-form-item label="用户名">
        <el-input v-model.trim="tmp.nickName"/>
      </el-form-item>
      <el-form-item label="电子邮箱">
        <el-input v-model.trim="tmp.email"/>
      </el-form-item>
      <el-form-item label="手机号码">
        <el-input v-model.trim="tmp.phone"/>
      </el-form-item>
      <p>最后登录时间：{{ tmp.lastLogin }}</p>
      <el-form-item>
        <el-button type="primary" @click="submit">更新</el-button>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script>
import {fetchUpdate} from "@/api/user";
import {setUserInfo} from "@/utils/auth";

export default {
  props: {
    user: {
      type: Object,
      default: () => {
        return {
          id: '',
          userName: '',
          nickName: '',
          email: '',
          phone: '',
          lastLogin: '',
          mfa: false
        }
      }
    }
  },
  data() {
    return {
      tmp: {}
    }
  },
  created() {
    this.tmp = this.user
  },
  methods: {
    submit() {
      const tmp = this.user
      fetchUpdate(tmp).then(() => {
        setUserInfo(this.tmp)
        this.$notify({
          message: '更新成功！',
          type: 'success'
        })
      })
    }
  }
}
</script>
