<template>
  <div class="app-container">
    <div v-if="user">
      <el-row :gutter="20">

        <el-col :span="6" :xs="24">
          <user-card :user="user"/>
        </el-col>

        <el-col :span="18" :xs="24">
          <account :user="user"/>
        </el-col>

      </el-row>
    </div>
  </div>
</template>

<script>
import {mapGetters} from 'vuex'
import UserCard from './components/UserCard'
import Account from './components/Account'
import {getUserInfo} from "@/utils/auth";

export default {
  name: 'Profile',
  components: {UserCard, Account},
  data() {
    return {
      user: {}
    }
  },
  computed: {
    ...mapGetters([
      'userinfo'
    ])
  },
  created() {
    this.getUser()
  },
  methods: {
    getUser() {
      this.user = getUserInfo()
    }
  }
}
</script>
