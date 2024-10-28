<template>
  <div class="login-container">
    <div class="logo-container el-header">
      <img src="../../assets/logo/kubackup-bar.png" class="logo">
    </div>
    <el-form ref="loginForm" :model="loginForm" :rules="loginRules" class="login-form" autocomplete="on"
             label-position="left">

      <div class="title-container">
        <h3 class="title">{{ $t('msg.login.title') }}</h3>
      </div>

      <el-form-item prop="username">
        <span class="svg-container">
          <svg-icon icon-class="user"/>
        </span>
        <el-input
          ref="username"
          v-model="loginForm.username"
          :placeholder="$t('msg.login.username')"
          name="username"
          type="text"
          tabindex="1"
          autocomplete="on"
          clearable
        />
      </el-form-item>

      <el-tooltip v-model="capsTooltip" :content="$t('msg.login.caps')" placement="right" manual>
        <el-form-item prop="password">
          <span class="svg-container">
            <svg-icon icon-class="password"/>
          </span>
          <el-input
            :key="passwordType"
            ref="password"
            v-model="loginForm.password"
            :type="passwordType"
            :placeholder="$t('msg.login.password')"
            name="password"
            tabindex="2"
            clearable
            autocomplete="on"
            @keyup.native="checkCapslock"
            @blur="capsTooltip = false"
            @keyup.enter.native="handleLogin"
          />
          <span class="show-pwd" @click="showPwd">
            <svg-icon :icon-class="passwordType === 'password' ? 'eye' : 'eye-open'"/>
          </span>
        </el-form-item>
      </el-tooltip>
      <div style="display: flex;justify-content: space-between;align-content: center;">
        <el-dropdown class="right-menu-item hover-effect" @command="switchLang" style="line-height: 50px;height: 50px;font-size: 15px">
          <span class="el-dropdown-link">
            {{ options[this.$i18n.locale] }}<i class="el-icon-arrow-down el-icon--right"></i>
          </span>
          <el-dropdown-menu>
            <el-dropdown-item command="zh-CN">中文</el-dropdown-item>
            <el-dropdown-item command="en-US">English</el-dropdown-item>
          </el-dropdown-menu>
        </el-dropdown>
        <p class="forget">
          <el-link type="primary" target="_blank" href="https://kubackup.cn/user_manual/user/#_4">
            {{ $t('msg.login.forgotPassword') }}
          </el-link>
        </p>
      </div>

      <el-button :loading="loading" type="primary" style="width:100%;margin-bottom:30px;"
                 @click.native.prevent="handleLogin">{{ $t('msg.login.login') }}
      </el-button>
    </el-form>
    <div class="el-footer footer">
      {{ $t('msg.login.support') }}:
      <el-link href="https://kubackup.cn" type="primary" :underline="false" target="_blank">酷备份 Kubackup</el-link>
      <p>{{ version }}</p>
    </div>
  </div>
</template>

<script>
import {title, title_en} from "@/settings";
import {fetchVersion} from "@/api/system";
import getPageTitle from "@/utils/get-page-title";

export default {
  name: 'Login',
  data() {
    const validateUsername = (rule, value, callback) => {
      if (!value) {
        callback(new Error(this.$t('msg.tips.usernameError')));
      } else {
        callback()
      }
    }
    const validatePassword = (rule, value, callback) => {
      if (value.length < 6) {
        callback(new Error(this.$t('msg.tips.passwordError')))
      } else {
        callback()
      }
    }
    return {
      title: title,
      title_en: title_en,
      version: '',
      loginForm: {
        username: '',
        password: '',
        code: ''
      },
      loginRules: {
        username: [{required: true, trigger: 'blur', validator: validateUsername}],
        password: [{required: true, trigger: 'blur', validator: validatePassword}]
      },
      options: {
        'zh-CN': '中文',
        'en-US': 'English'
      },
      passwordType: 'password',
      capsTooltip: false,
      loading: false,
      redirect: undefined,
      otherQuery: {}
    }
  },
  watch: {
    $route: {
      handler: function (route) {
        const query = route.query
        if (query) {
          this.redirect = query.redirect
          this.otherQuery = this.getOtherQuery(query)
        }
      },
      immediate: true
    }
  },
  mounted() {
    if (this.loginForm.username === '') {
      this.$refs.username.focus()
    } else if (this.loginForm.password === '') {
      this.$refs.password.focus()
    }
  },
  created() {
    this.getVersion()
  },
  methods: {
    checkCapslock(e) {
      const {key} = e
      this.capsTooltip = key && key.length === 1 && (key >= 'A' && key <= 'Z')
    },
    getVersion() {
      fetchVersion().then(res => {
        const v = res.data;
        this.version = v.version
      })
    },
    switchLang(cmd) {
      this.$i18n.locale = cmd
      localStorage.setItem('locale', this.$i18n.locale)
      document.title = getPageTitle(this.$t(this.$route.meta.title))
    },
    showPwd() {
      if (this.passwordType === 'password') {
        this.passwordType = ''
      } else {
        this.passwordType = 'password'
      }
      this.$nextTick(() => {
        this.$refs.password.focus()
      })
    },
    handleLogin() {
      this.$refs.loginForm.validate(valid => {
        if (valid) {
          this.loading = true
          this.$store.dispatch('user/login', this.loginForm)
            .then(() => {
              this.loading = false
              this.$router.push({path: this.redirect || '/', query: this.otherQuery})
            })
            .catch((err) => {
              this.loading = false
              if (err === 'mfa') {
                this.handleCode()
              }
            })
        } else {
          return false
        }
      })
    },
    getOtherQuery(query) {
      return Object.keys(query).reduce((acc, cur) => {
        if (cur !== 'redirect') {
          acc[cur] = query[cur]
        }
        return acc
      }, {})
    },
    // 验证码登录
    handleCode() {
      this.$prompt(this.$t('msg.tips.captchaMsg'), this.$t('msg.login.captcha'), {
        cancelButtonText: this.$t('msg.cancel'),
        inputPattern: /^\d{6,}$/,
        inputErrorMessage: this.$t('msg.tips.captchaError')
      }).then(({value}) => {
        this.loginForm.code = value
        this.handleLogin()
      })
    }
  }
}
</script>

<style lang="scss">
@import "../dashboard/src/styles/variables";

$bg: $menuBg;
$cursor: $menuText;
$dark_gray: #000;
$light_gray: $menuHover;


@supports (-webkit-mask: none) and (not (cater-color: $cursor)) {
  .login-container .el-input input {
    color: $cursor;
  }
}

.login-container {

  min-height: 100%;
  width: 100%;
  background-color: $bg;
  overflow: hidden;

  .footer {
    font-size: 15px;
    position: absolute;
    bottom: 20px;
    left: 50%;
    transform: translateX(-50%);
    text-align: center;

  }

  .logo-container {
    padding: 30px 0 0 30px;

    .logo {
      height: 50px;
      vertical-align: middle;
    }
  }

  .el-input {
    display: inline-block;
    height: 47px;
    width: 85%;

    input {
      background: transparent;
      border: 0px;
      -webkit-appearance: none;
      border-radius: 0px;
      padding: 12px 5px 12px 15px;
      color: $cursor;
      height: 47px;
      caret-color: $cursor;

      &:-webkit-autofill {
        box-shadow: 0 0 0px 1000px $bg inset !important;
        -webkit-text-fill-color: $cursor !important;
      }
    }
  }

  .el-form-item {
    border: 1px solid rgba(255, 255, 255, 0.1);
    background: $bg;
    border-radius: 5px;
    color: $cursor;
  }

  .login-form {
    box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
    position: relative;
    width: 520px;
    max-width: 100%;
    background-color: $light_gray;
    padding: 20px;
    border-radius: 20px;
    margin: 110px auto;
    overflow: hidden;

    .el-dropdown-link {
      cursor: pointer;
      color: #409EFF;
    }

    .el-icon-arrow-down {
      font-size: 12px;
    }
    .forget {
      text-align: right;
    }
  }

  .svg-container {
    padding: 6px 5px 6px 15px;
    color: $dark_gray;
    vertical-align: middle;
    width: 30px;
    display: inline-block;
  }

  .title-container {
    position: relative;

    .title {
      font-size: 26px;
      color: $menuActiveText;
      margin: 0px auto 40px auto;
      text-align: center;
      font-weight: bold;
    }
  }

  .show-pwd {
    position: absolute;
    right: 10px;
    top: 7px;
    font-size: 16px;
    color: $dark_gray;
    cursor: pointer;
    user-select: none;
  }

  @media only screen and (max-width: 470px) {
    .thirdparty-button {
      display: none;
    }
  }
}
</style>

