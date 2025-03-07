<template>
  <el-card>
    <div slot="header" class="clearfix">
      <span>{{ $t('msg.title.profile') }}</span>
    </div>

    <div class="user-profile">
      <div class="box-center">
        <pan-thumb :height="'100px'" :width="'100px'" :hoverable="false">
          <div>{{ $t('msg.hello') }}</div>
          {{ user.nickName }}
        </pan-thumb>
      </div>
      <div class="box-center">
        <div class="user-name text-center">{{ user.userName }}</div>
      </div>
      <div class="box-center">
        <el-button type="primary" @click="repwdHandler">{{ $t('msg.changePassword') }}</el-button>
      </div>
      <div class="box-center">
        <el-switch
          v-model="this.user.mfa"
          :active-text="$t('msg.twoFactorAuth')" @change="mfaChangeHandler">
        </el-switch>
      </div>
      <div class="box-center" v-if="mfa">
        <p>{{ $t('msg.scanQrCodeTip') }}<el-link type="success" target="_blank" href="https://kubackup.cn/user_manual/user/#_2">otp{{ $t('msg.application') }}</el-link>{{ $t('msg.getVerificationCode') }}
        </p>
        <img @click="getQrcode" v-if="mfaQrcode" :src="mfaQrcode" class="qrcode" alt="qrcode">
        <p class="secret">{{ $t('msg.secretKey') }}：{{ otpInfo.secret }}</p>
        <el-input :placeholder="$t('msg.pleaseInput') + $t('msg.verificationCode')" v-model="otpInfo.code" class="input-with-select mfacode">
          <el-button slot="append" @click="bindOtp">{{ $t('msg.bind') }}</el-button>
        </el-input>
      </div>
    </div>
    <el-dialog
      :title="$t('msg.changePassword')"
      :visible.sync="dialogFormVisible"
    >
      <el-form ref="dataForm" :rules="rules" :model="temp" label-position="left" label-width="120px">
        <el-form-item :label="$t('msg.oldPassword')" prop="oldPassword">
          <el-input v-model="temp.oldPassword" type="password" clearable/>
        </el-form-item>
        <el-form-item :label="$t('msg.newPassword')" prop="password">
          <el-input v-model="temp.password" type="password" clearable/>
        </el-form-item>
        <el-form-item :label="$t('msg.confirmPassword')" prop="confirmPassword">
          <el-input v-model="temp.confirmPassword" type="password" clearable/>
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogFormVisible = false">
          {{ $t('msg.cancel') }}
        </el-button>
        <el-button
          type="primary"
          @click="repwd"
        >
          {{ $t('msg.confirm') }}
        </el-button>
      </div>
    </el-dialog>
  </el-card>
</template>

<script>
import PanThumb from '@/components/PanThumb'
import {fetchBindOtp, fetchDel, fetchDeleteOtp, fetchOtp, fetchRePwd} from "@/api/user";
import {setUserInfo} from "@/utils/auth";

export default {
  components: {PanThumb},
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
      dialogFormVisible: false,
      mfa: false,
      mfaQrcode: '',
      otpInfo: {
        secret: '',
        code: '',
        interval: 0
      },
      temp: {
        oldPassword: "",
        password: "",
        confirmPassword: '',
      },
      rules: {
        oldPassword: [{required: true, message: this.$t('msg.tips.emptyError'), trigger: 'blur'}],
        confirmPassword: [{required: true, validator: validatePassword, trigger: 'blur'}],
        password: [{required: true, message: this.$t('msg.tips.emptyError'), trigger: 'blur'}],
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
              title: this.$t('msg.title.notice'),
              message: res.data
            })
          }).finally(() => {
            this.dialogFormVisible = false
          })
        }
      })
    },
    mfaChangeHandler(value) {
      if (value) {
        this.mfa = true
        this.getQrcode()
      } else {
        this.$confirm(this.$t('msg.confirmCloseTwoFactorAuth'), this.$t('msg.closeAuth'), {
          type: 'warning'
        }).then(() => {
          this.otpInfo = {
            secret: '',
            code: '',
            interval: 0
          }
          fetchDeleteOtp().then(res => {
            this.user.mfa = false
            setUserInfo(this.user)
            this.mfa = false
            this.$notify.success({
              title: this.$t('msg.title.notice'),
              message: res.data
            })
          })
        })
      }
    },
    getQrcode() {
      fetchOtp().then(res => {
        const data = res.data
        this.otpInfo.secret = data.secret
        this.otpInfo.interval = data.interval
        this.mfaQrcode = data.qrImg
      })
    },
    bindOtp() {
      if (this.otpInfo.code === '') {
        this.$notify.error({
          title: this.$t('msg.err'),
          message: this.$t('msg.verificationCodeCannotBeEmpty')
        })
      }
      fetchBindOtp(this.otpInfo).then(res => {
        this.$notify.success({
          title: this.$t('msg.title.notice'),
          message: res.data
        })
        this.user.mfa = true
        setUserInfo(this.user)
        this.mfa = false
      })
    }
  }
}
</script>

<style lang="scss" scoped>
@import "../dashboard/src/styles/variables";

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

  .qrcode {
    display: block;
    margin: auto;
    width: 200px;
    height: 200px;
  }

  .secret {
    display: block;
    margin: auto;
    font-size: 15px;
  }

  .mfacode {
    margin-top: 10px;
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
