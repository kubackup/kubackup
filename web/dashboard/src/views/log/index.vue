<template>
  <div class="app-container">
    <div class="handle-search">
      <el-form :model="listQuery" inline @submit.native.prevent>
        <el-form-item :label="$t('msg.title.operator')">
          <el-input v-model="listQuery.operator" :placeholder="$t('msg.title.operator')" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item :label="$t('msg.title.operationAction')">
          <el-select v-model="listQuery.operation" class="handle-select mr5" clearable :placeholder="$t('msg.pleaseSelect')">
            <el-option
              v-for="(item, index) in [{value: '', name: $t('msg.all')}].concat(operationList)"
              :key="index"
              :label="item.name"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('msg.title.resource')">
          <el-select v-model="listQuery.url" class="handle-select mr5" clearable :placeholder="$t('msg.pleaseSelect')">
            <el-option
              v-for="(item, index) in [{value: '', name: $t('msg.all')}].concat(resourceList)"
              :key="index"
              :label="item.name"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('msg.title.data')">
          <el-input v-model="listQuery.data" placeholder="" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="el-icon-search" @click="handleFilter">{{ $t('msg.search') }}</el-button>
        </el-form-item>
      </el-form>
    </div>
    <el-table v-loading="listLoading" :data="list" border fit highlight-current-row style="width: 100%">
      <el-table-column prop="id" align="center" label="ID"/>
      <el-table-column prop="createdAt" :formatter="dateFormat" align="center" :label="'createdAt' | i18n"/>
      <el-table-column prop="operator" align="center" :label="$t('msg.title.operator')"/>
      <el-table-column prop="operation" align="center" :label="$t('msg.title.operationAction')">
        <template slot-scope="{row}">
          <el-tag :type="filterOper(row.operation).color">
            {{ filterOper(row.operation).name }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="url" align="center" :label="$t('msg.title.resource')">
        <template slot-scope="{row}">
          {{ filterResource(row.url).name }}
        </template>
      </el-table-column>
      <el-table-column prop="url" align="center" :label="$t('msg.title.address')"/>
      <el-table-column prop="data" align="center" :label="$t('msg.title.data')">
        <template slot-scope="{row}">
          <i v-if="row.data" class="el-icon-document" @click="handleInfo(row)"></i>
        </template>
      </el-table-column>
    </el-table>
    <pagination
      v-show="total > 0"
      :total="total"
      :page.sync="listQuery.pageNum"
      :limit.sync="listQuery.pageSize"
      :autoScroll="false"
      @pagination="getList"
    />
    <el-dialog :title="$t('msg.title.dataDetails')" :visible.sync="dialogFormVisible" top="5vh">
      <el-input class="json-text" type="textarea" readonly autosize v-model="flowJSON"></el-input>
    </el-dialog>
  </div>
</template>

<script>
import {fetchLogs} from '@/api/dashboard'
import {dateFormat} from "@/utils";
import Pagination from '@/components/Pagination'

export default {
  name: 'LogList',
  components: {Pagination},
  data() {
    return {
      operationList: [
        {name: this.$t('msg.status.add'), value: 'post', color: 'success'},
        {name: this.$t('msg.status.modify'), value: 'put', color: 'primary'},
        {name: this.$t('msg.status.delete'), value: 'delete', color: 'danger'},
      ],
      resourceList: [
        {name: this.$t('msg.status.login'), value: 'login'},
        {name: this.$t('msg.status.repository'), value: 'repository'},
        {name: this.$t('msg.status.plan'), value: 'plan'},
        {name: this.$t('msg.status.user'), value: 'user'},
        {name: this.$t('msg.status.backup'), value: 'backup'},
        {name: this.$t('msg.status.restore'), value: 'restore'},
        {name: this.$t('msg.status.policy'), value: 'policy'},
        {name: this.$t('msg.status.maintenance'), value: 'restic'}
      ],
      listLoading: false,
      listQuery: {
        operator: '',
        operation: '',
        url: '',
        data: '',
        pageNum: 1,
        pageSize: 10
      },
      list: [],
      total: 0,
      dialogFormVisible: false,
      flowJSON: {}
    }
  },
  created() {
    this.getList()
  },
  methods: {
    dateFormat(row, column, cellValue, index) {
      return dateFormat(cellValue, 'yyyy-MM-dd hh:mm:ss')
    },
    filterOper(code) {
      return this.operationList.find(item => item.value === code) || {name: '', value: code}
    },
    filterResource(code) {
      return this.resourceList.find(item => code.includes(item.value)) || {name: '', value: code}
    },
    handleFilter() {
      this.listQuery.pageNum = 1
      this.getList()
    },
    handleInfo(row) {
      this.dialogFormVisible = true
      this.flowJSON = {}
      this.flowJSON = JSON.stringify(JSON.parse(row.data), null, "    ")
    },
    getList() {
      this.listLoading = true
      fetchLogs(this.listQuery).then(response => {
        this.list = response.data.items
        this.total = response.data.total
      }).finally(() => {
        this.listLoading = false
      })
    }
  }
}
</script>

<style lang="scss" scoped>
.json-text {
  color: #000000;
  font-weight: bold;
}
</style>
