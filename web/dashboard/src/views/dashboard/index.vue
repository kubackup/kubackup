<template>
  <div class="dashboard-container">
    <span class="duration">本次统计时间：{{ dateformat(backupinfo.time) }} 耗时：{{ backupinfo.duration }}</span>
    <i class="refresh-btn" :class="refresh_icon" @click="doGetAllRepoStats"/>
    <el-row :gutter="40" class="panel-group">
      <el-col :xs="12" :sm="12" :lg="8" class="card-panel-col">
        <panel
          :start-val="1"
          :end-val="backupinfo.dataDayNum"
          text="数据保护天数"
          :suffix="backupinfo.dataDayUnit"
          icon="el-icon-time"
          icon-color="icon-green"
        />
      </el-col>
      <el-col :xs="12" :sm="12" :lg="8" class="card-panel-col">
        <panel
          :start-val="1"
          :end-val="backupinfo.dataSizeStrNum"
          text="数据量"
          :suffix="backupinfo.dataSizeStrUnit"
          icon="el-icon-s-data"
          icon-color="icon-blue"
        />
      </el-col>
      <el-col :xs="12" :sm="12" :lg="8" class="card-panel-col">
        <panel :start-val="1" :end-val="backupinfo.fileTotal" text="文件数量" icon-color="icon-red" icon="el-icon-files"/>
      </el-col>
      <el-col :xs="12" :sm="12" :lg="8" class="card-panel-col">
        <panel
          :start-val="1"
          :end-val="backupinfo.snapshotsNum"
          text="快照数量"
          icon-color="icon-blue"
          icon="el-icon-s-flag"
        />
      </el-col>
      <el-col :xs="12" :sm="12" :lg="8" class="card-panel-col">
        <panel
          :countto="false"
          :cvalue="planinfo.runningCount+' / '+planinfo.total"
          text="任务数量"
          icon="el-icon-tickets"
          icon-color="icon-red"
        />
      </el-col>
      <el-col :xs="12" :sm="12" :lg="8" class="card-panel-col">
        <panel
          :countto="false"
          :cvalue="repoinfo.runningCount+' / '+repoinfo.total"
          text="仓库数量"
          icon="el-icon-coin"
          icon-color="icon-green"
        />
      </el-col>
    </el-row>
    <el-row :gutter="40" class="panel-group">
      <el-col :xs="12" :sm="12" :lg="8" class="card-panel-col">
        <pie_chart title="备份数据量统计" :cdata="chartsize" theme="shine" seriesname="单位：GB"/>
      </el-col>
      <el-col :xs="12" :sm="12" :lg="8" class="card-panel-col">
        <pie_chart title="备份文件数量统计" :cdata="chartfile" theme="shine"/>
      </el-col>
      <el-col :xs="12" :sm="12" :lg="8" class="card-panel-col">
        <pie_chart title="备份快照数量统计" :cdata="chartsnap" theme="shine"/>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import Panel from './panel/panel'
import pie_chart from './panel/PieChart'
import {fetchDoGetAllRepoStats, fetchIndex} from '@/api/dashboard'
import resize from './mixins/resize'
import {dateFormat} from "@/utils";

export default {
  name: 'Dashboard',
  components: {Panel, pie_chart},
  mixins: [resize],
  data() {
    return {
      backupinfo: {},
      planinfo: {},
      repoinfo: {},
      chartsize: [],
      chartfile: [],
      chartsnap: [],
      refresh_icon: 'el-icon-refresh-left'
    }
  },
  created() {
    this.InitData()
  },
  methods: {
    dateformat(v) {
      return dateFormat(v, 'yyyy-MM-dd hh:mm:ss')
    },
    InitData() {
      fetchIndex().then(res => {
        var backupinfo = res.data.backupInfo
        var planinfo = res.data.planInfo
        var repoinfo = res.data.repositoryInfo
        backupinfo.dataDayNum = Number(backupinfo.dataDay.split(' ')[0])
        backupinfo.dataDayUnit = backupinfo.dataDay.split(' ')[1]
        backupinfo.dataSizeStrNum = Number(backupinfo.dataSizeStr.split(' ')[0])
        backupinfo.dataSizeStrUnit = backupinfo.dataSizeStr.split(' ')[1]
        this.backupinfo = backupinfo
        this.planinfo = planinfo
        this.repoinfo = repoinfo
        this.loadChart(res.data.backupInfos)
      })
    },
    doGetAllRepoStats() {
      this.refresh_icon = 'el-icon-loading'
      fetchDoGetAllRepoStats().then(res => {
        this.$notify.success(
          {
            message: '正在统计中，请稍后刷新页面查看...',
            title: '提示'
          })
      }).finally(() => {
        this.refresh_icon = 'el-icon-refresh-left'
      })
    },
    loadChart(backupInfos) {
      const len = backupInfos.length
      let i = 0
      for (; i < len; i++) {
        const info = backupInfos[i]
        this.chartsize.push({value: (info.dataSize / 1000 / 1000 / 1000).toFixed(2), name: info.repositoryName})
        this.chartfile.push({value: info.fileTotal, name: info.repositoryName})
        this.chartsnap.push({value: info.snapshotsNum, name: info.repositoryName})
      }
    }
  }
}
</script>
<style lang="scss" scoped>
@import "src/styles/variables";

.dashboard-container {
  padding: 18px;

  .duration {

  }

  .refresh-btn {
    margin-left: 10px;
    cursor: pointer;
    color: $light-blue;
  }

  .panel-group {
    margin-top: 10px;

    .card-panel-col {
      margin-bottom: 32px;
    }
  }

}

</style>
