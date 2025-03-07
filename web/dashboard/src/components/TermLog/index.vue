<template>
  <div class="vue-terminal">
    <div v-if="showHeader" class="terminal-header">
      <h4>{{ title }}</h4>
      <ul class="shell-dots">
        <li class="shell-dots-red" />
        <li class="shell-dots-yellow" />
        <li class="shell-dots-green" />
      </ul>
      <div class="terminal-header-right">
        <span v-if="status!==0" :class="formatStatus(status).color">{{ formatStatus(status).name }}</span>
        <span style="margin-left: 10px">{{ $t('msg.title.autoScroll') }} <el-switch v-model="autoScroll" /></span>
      </div>
    </div>

    <div>
      <div ref="terminalWindow" class="terminal-window">
        <p v-for="(item, index) in messageList" :key="index">
          <span v-if="item.time">{{ dateFormat(item.time) }}</span>
          <span v-if="item.level" class="level" :class="formatLevel(item.level).color">{{
            formatLevel(item.level).name
          }}</span>
          <code>{{ item.text }}</code>
        </p>
        <p ref="terminalLastLine" class="terminal-last-line" />
      </div>
    </div>
  </div>
</template>
<script>

import { dateFormat } from '@/utils'
import { LoglevelList } from '@/consts'

export default {
  name: 'Terminal',
  props: {
    title: {
      required: false,
      type: String,
      default: 'Terminal'
    },
    status: {
      required: false,
      type: Number,
      default: 0
    },
    showHeader: {
      required: false,
      type: Boolean,
      default: false
    },
    autoScroll: {
      required: false,
      type: Boolean,
      default: true
    },
    init: {
      required: false,
      type: Boolean,
      default: true
    },
    data: {
      required: true,
      type: Array,
      default: []
    }
  },
  data() {
    return {
      messageList: [],
      levelList: LoglevelList,
      statusList: [
        { code: 1, name: this.$t('msg.status.getting'), color: 'info' },
        { code: 2, name: this.$t('msg.status.normal'), color: 'success' },
        { code: 3, name: this.$t('msg.status.error'), color: 'error' }
      ]
    }
  },
  watch: {
    data(val) {
      if (val === null) {
        val = []
      }
      let datas = []
      if (!this.init && (val.length > this.messageList.length)) {
        datas = val.slice(this.messageList.length, val.length)
      } else {
        this.messageList = []
        datas = val
      }
      datas.forEach(v => {
        const end = this.messageList[this.messageList.length - 1]
        if (v.clear && end !== undefined && end.clear) {
          this.messageList.pop()
        }
        this.pushToList(v)
      })
    }
  },

  methods: {
    dateFormat(cellValue) {
      return dateFormat(cellValue, 'yyyy-MM-dd hh:mm:ss')
    },
    formatStatus(code) {
      let res = this.statusList.find(item => item.code === code)
      if (!res) {
        res = { code: 1, name: 'Info', color: 'info' }
      }
      return res
    },
    formatLevel(code) {
      let res = this.levelList.find(item => item.code === code)
      if (!res) {
        res = { code: 1, name: 'Info', color: 'info' }
      }
      return res
    },
    pushToList(message) {
      this.messageList.push(message)
      if (this.autoScroll) {
        this.autoScrollHandler()
      }
    },
    autoScrollHandler() {
      this.$nextTick(() => {
        this.$refs.terminalWindow.scrollTop = this.$refs.terminalLastLine.offsetTop
      })
    }
  }
}
</script>

<style scoped lang="scss">
@import "src/styles/variables";

.vue-terminal {
  position: relative;
  width: 100%;
  border-radius: 4px;
  color: white;
  max-height: 600px;
}

.vue-terminal .terminal-window {
  overflow: auto;
  z-index: 1;
  max-height: 500px;
  background-color: $menuBg;
  min-height: 140px;
  padding: 10px;
  font-weight: normal;
  font-family: Monaco, Menlo, Consolas, monospace;
  color: #000000;

  p {
    overflow-wrap: break-word;
    word-break: break-all;
    font-size: 13px;

    .level {
      margin-left: 4px;
      padding: 2px 3px;
    }

    code {
      margin-left: 4px;
      display: inline;
      font-family: Monaco, Menlo, Consolas, monospace;
      white-space: pre-wrap;
    }
  }
}

.vue-terminal .terminal-window .loading {
  display: inline-block;
  width: 0;
  overflow: hidden;
  overflow-wrap: normal;
  animation: load 1.2s step-end infinite;
  -webkit-animation: load 1.2s step-end infinite;
}

.vue-terminal .terminal-window .cursor {
  margin: 0;
  background-color: white;
  animation: blink 1s step-end infinite;
  -webkit-animation: blink 1s step-end infinite;
  margin-left: -5px;
}

.info {
  background: #2980b9;
  color: #ffffff;
}

.warning {
  background: #f39c12;
  color: #ffffff;
}

.success {
  background: #27ae60;
  color: #ffffff;
}

.error {
  background: #c0392b;
  color: #ffffff;
}

.terminal-header ul.shell-dots li {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 6px;
  margin-left: 6px
}

.terminal-header ul .shell-dots-red {
  background-color: rgb(200, 48, 48);
}

.terminal-header ul .shell-dots-yellow {
  background-color: rgb(247, 219, 96);
}

.terminal-header ul .shell-dots-green {
  background-color: rgb(46, 201, 113);
}

.terminal-header {
  background-color: rgb(149, 149, 152);
  text-align: center;
  padding: 2px;
  border-top-left-radius: 4px;
  border-top-right-radius: 4px
}

.terminal-header h4 {
  font-size: 14px;
  margin: 5px;
  letter-spacing: 1px;
}

.terminal-header .terminal-header-right {
  position: absolute;
  font-size: 14px;
  top: 6px;
  right: 8px;
  margin: 0;
}

.terminal-header ul.shell-dots {
  position: absolute;
  top: 6px;
  left: 8px;
  padding-left: 0;
  margin: 0;
}

.terminal-last-line {
  font-size: 0;
  word-spacing: 0;
  letter-spacing: 0;
}

</style>
