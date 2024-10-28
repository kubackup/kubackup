<template>
  <div class="app-container">
    <div class="handle-search">
      <el-form :model="listQuery" inline @submit.native.prevent>
        <el-form-item label="操作员">
          <el-input v-model="listQuery.operator" placeholder="操作员" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item label="操作">
          <el-select v-model="listQuery.operation" class="handle-select mr5" clearable :placeholder="$t('msg.pleaseSelect')">
            <el-option
              v-for="(item, index) in [{value: '', name: '所有'}].concat(operationList)"
              :key="index"
              :label="item.name"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="资源">
          <el-select v-model="listQuery.url" class="handle-select mr5" clearable :placeholder="$t('msg.pleaseSelect')">
            <el-option
              v-for="(item, index) in [{value: '', name: '所有'}].concat(resourceList)"
              :key="index"
              :label="item.name"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="数据">
          <el-input v-model="listQuery.data" placeholder="" class="filter-item" clearable/>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="el-icon-search" @click="handleFilter">查询</el-button>
        </el-form-item>
      </el-form>
    </div>
    <el-table v-loading="listLoading" :data="list" border fit highlight-current-row style="width: 100%">
      <el-table-column prop="id" align="center" label="ID"/>
      <el-table-column prop="createdAt" :formatter="dateFormat" align="center" :label="'createdAt' | i18n"/>
      <el-table-column prop="operator" align="center" label="操作员"/>
      <el-table-column prop="operation" align="center" label="操作">
        <template slot-scope="{row}">
          <el-tag :type="filterOper(row.operation).color">
            {{ filterOper(row.operation).name }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="url" align="center" label="资源">
        <template slot-scope="{row}">
          {{ filterResource(row.url).name }}
        </template>
      </el-table-column>
      <el-table-column prop="url" align="center" label="地址"/>
      <el-table-column prop="data" align="center" label="数据">
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
    <el-dialog title="数据详情" :visible.sync="dialogFormVisible" top="5vh">
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
        {name: '新增', value: 'post', color: 'success'},
        {name: '修改', value: 'put', color: 'primary'},
        {name: '删除', value: 'delete', color: 'danger'},
      ],
      resourceList: [
        {name: '登录', value: 'login'},
        {name: '存储库', value: 'repository'},
        {name: '备份计划', value: 'plan'},
        {name: '用户', value: 'user'},
        {name: '执行备份', value: 'backup'},
        {name: '执行数据恢复', value: 'restore'},
        {name: '清理策略', value: 'policy'},
        {name: '数据维护', value: 'restic'}
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
