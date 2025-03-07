<template>
  <div class="errPage-container">
    <el-button icon="el-icon-arrow-left" class="pan-back-btn" @click="back">
      {{ $t('msg.back') }}
    </el-button>
    <el-row>
      <el-col :span="12">
        <h1 class="text-jumbo">
          Oops!
        </h1>
        <h2>{{ $t('msg.noPermission') }}</h2>
        <ul class="list-unstyled">
          <li>{{ $t('msg.orYouCanGoTo') }}:</li>
          <li class="link-type">
            <router-link to="/dashboard">
              {{ $t('msg.returnHome') }}
            </router-link>
          </li>
        </ul>
      </el-col>
      <el-col :span="12"><span style="font-size: 200px;color: #1482f0">403</span>
      </el-col>
    </el-row>
  </div>
</template>

<script>
export default {
  name: 'Page403',
  data() {
    return {
    }
  },
  methods: {
    back() {
      if (this.$route.query.noGoBack) {
        this.$router.push({ path: '/dashboard' })
      } else {
        this.$router.go(-1)
      }
    }
  }
}
</script>

<style lang="scss" scoped>
  .errPage-container {
    width: 800px;
    max-width: 100%;
    margin: 100px auto;
    .pan-back-btn {
      background: #1482f0;
      color: #fff;
      border: none!important;
    }
    .pan-gif {
      margin: 0 auto;
      display: block;
    }
    .text-jumbo {
      font-size: 60px;
      font-weight: 700;
      color: #484848;
    }
  }
</style>
