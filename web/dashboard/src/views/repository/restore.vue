<template>
  <div class="app-container">
    <div class="handle-search">
      <el-form :model="listQuery" inline @submit.native.prevent>
        <el-form-item :label="$t('msg.select')">
          <el-cascader
            v-model="pathQuery"
            :options="hostList"
            :props="{ expandTrigger: 'hover' }"
            clearable
            :placeholder="$t('msg.pleaseSelect')"
            filterable
            separator=" => "
            style="width: 800px"
            @change="handleSearch"/>
        </el-form-item>
        <el-form-item>
          <el-date-picker
            v-model="listQuery.date"
            type="date"
            :placeholder="$t('msg.pleaseSelect')"
            @change="handleSearch">
          </el-date-picker>
        </el-form-item>
      </el-form>
    </div>
    <div>
      <el-row :gutter="40" class="panel-group">
        <el-col :xs="8" :sm="8" :lg="8" class="card-panel-col">
          <el-card class="box-card">
            <div slot="header">
              <p>{{ $t('msg.host') }}：{{ listQuery.host + (list.length > 0 ? '(' + list[0].username + ')' : '') }}</p>
              <p>{{ $t('msg.path') }}：{{ listQuery.path }}</p>
              <p>{{ $t('msg.total') }}：{{ total }}</p>
            </div>
            <el-collapse v-model="activeName" accordion>
              <el-collapse-item :title="snaps.name" :name="snaps.name" :key="i" v-for="(snaps, i) in snapList">
                <el-timeline>
                  <el-timeline-item
                    v-for="(item, index) in snaps.list"
                    :key="index"
                    :timestamp="item.time|goDatToDateString"
                    type="primary"
                    icon="el-icon-success"
                    size="large"
                    placement="top"
                  >
                    <div class="timeline">
                      <span class="snap" @click="loadSnapFiles(item)">{{ item.short_id }}</span>
                      <el-button
                        class="restore_btn"
                        type="text"
                        size="mini"
                        @click="openRestoreOpt(item.short_id)">
                        {{ $t('msg.operation.restore') }}
                      </el-button>
                    </div>
                  </el-timeline-item>
                </el-timeline>
              </el-collapse-item>
            </el-collapse>
            <p v-if="noMore" style="font-size: 20px; text-align: center; color: #bbbbbb">{{ $t('msg.tips.noMore') }}</p>
            <div v-else style="text-align: center;">
              <el-button :loading="listLoading" type="info" plain @click="getMoreList">{{ $t('msg.loadMore') }}</el-button>
            </div>
          </el-card>
        </el-col>
        <el-col :xs="16" :sm="16" :lg="16" class="card-panel-col">
          <el-card class="box-card">
            <div>
              <el-input :placeholder="$t('msg.tips.searchInfo')+'：data/test/avator.png'" v-model="fileSearch.name"
                        clearable
                        @clear="searchFile"
                        class="input-with-select">
                <el-select v-model="fileSearch.type" slot="prepend" :placeholder="$t('msg.pleaseSelect')">
                  <el-option
                    v-for="(item, index) in searchType"
                    :key="index"
                    :label="item.name"
                    :value="item.code"
                  />
                </el-select>
                <el-button slot="append" icon="el-icon-search" @click="searchFile"></el-button>
              </el-input>
            </div>
            <el-tree
              class="file-tree"
              :data="filedata.children"
              node-key="id"
              v-loading="treeLoading"
              accordion
              expand-on-click-node @node-expand="getFiles">
              <span class="custom-tree-node" slot-scope="{ node, data }" @click="moreClick(data,node)">
                <div class="file-title">
                  <i v-if="data.type==='dir'" class="el-icon-folder"/>
                  <i v-if="data.type==='btn'" class="el-icon-more-outline"/>
                  <i v-if="data.type==='file'" class="el-icon-document"/>
                  <span style="margin-left: 5px">{{ node.label }}</span>
                </div>
                <span>
                  <span style="margin-right: 10px; font-size: 13px; display: inline">{{ data.size }}</span>
                  <i class="el-icon-loading" type="primary" v-if="data.loading"/>
                  <el-button
                    v-if="data.isMore===0 && !data.loading"
                    type="text"
                    size="mini"
                    @click="() => restoreFileHandler(data)">
                    {{ $t('msg.operation.restore') }}
                  </el-button>
                </span>
              </span>
            </el-tree>
          </el-card>
        </el-col>
      </el-row>
    </div>
    <el-dialog
      :title="$t('msg.title.restoreOptions')"
      :visible.sync="dialogFormVisible"
    >
      <el-form ref="dataForm" label-position="left" label-width="150px">
        <el-form-item :label="$t('msg.title.restoreTo')+'：'" prop="path">
          <el-input v-model="restoreOpt.dirCur" disabled>
            <el-button slot="append" @click="openDirSelect()">{{ $t('msg.select') }}</el-button>
          </el-input>
          <span style="color: red">{{ $t('restore.defaultPathTip', {path: listQuery.path}) }}</span>
        </el-form-item>
        <el-form-item :label="$t('msg.title.finalPath')+'：'" prop="path">
          <span>{{ restoreOpt.dist }}</span>
        </el-form-item>
        <el-form-item :label="$t('msg.title.verify')">
          <el-switch
            v-model="restoreOpt.verify">
          </el-switch>
          <p style="color: red">{{ $t('restore.verifyTip') }}</p>
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogFormVisible = false">
          {{ $t('msg.cancel') }}
        </el-button>
        <el-button
          type="primary"
          @click="restoreSnapHandler()"
        >
          {{ $t('msg.confirm') }}
        </el-button>
      </div>
    </el-dialog>
    <el-dialog
      :title="$t('msg.title.folderSelect')"
      :visible.sync="dialogDirVisible"
    >
      <div>
        <el-breadcrumb separator="/">
          <el-breadcrumb-item class="breadcrumb-item" v-for="(item, index) in getDirSpea()" :key="index">
            <span class="title" @click="lsDir(item.path,true)">{{ item.name }}</span>
          </el-breadcrumb-item>
        </el-breadcrumb>
        <div class="filenodes">
          <span class="custom-tree-node filenode" v-for="(item, index) in dirList" :key="index" v-if="item.isDir"
                :class="{active : restoreOpt.dirCur === item.path}"
                @dblclick.prevent="lsDir(item.path,item.isDir)"
                @click="selectDir(item.path,item.isDir)">
          <div class="file-title">
            <i v-if="item.isDir" class="el-icon-folder"/>
            <span style="margin-left: 5px;user-select: none;">{{ item.name }}</span>
          </div>
          <span>
            <el-button
              type="text"
              class="confirmbtn"
              size="mini"
              @click="confirmDirSelect(item.path)">
                    {{ $t('msg.confirm') }}
                  </el-button>
          </span>
        </span>
        </div>
      </div>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogDirVisible = false">
          {{ $t('msg.cancel') }}
        </el-button>
        <el-button
          type="primary"
          @click="confirmDirSelect()">
          {{ $t('msg.confirm') }}
        </el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import {fetchLsList, fetchParmsList, fetchSearchList, fetchSnapshotsList} from '@/api/repository'
import {dateFormat} from "@/utils";
import {fetchRestore} from "@/api/task";
import {fetchLs} from "@/api/system";

export default {
  name: 'Restore',
  data() {
    return {
      list: [],
      total: 0,
      snapList: [],
      tagList: [],
      hostList: [],
      dialogFormVisible: false,
      dialogDirVisible: false,
      dirList: [],
      restoreOpt: {
        dirCur: '/',
        snapid: '',
        include: '',
        dist: '',
        verify: false
      },
      listLoading: false,
      treeLoading: false,
      noMore: true,
      curSnap: {},
      activeName: '0',
      searchType: [
        {code: 0, name: this.$t('msg.repository')},
        {code: 1, name: this.$t('msg.snapshot')}
      ],
      fileSearch: {
        type: 0,
        name: '',
        pageNum: 1,
        pageSize: 20,
      },
      pathQuery: [],
      listQuery: {
        id: 0,
        path: '',
        date: '',
        host: '',
        tags: '',
        pageNum: 1,
        pageSize: 100
      },
      filedata: {
        label: 'root',
        path: '',
        type: 'dir',
        isMore: 3,
        pageNum: 1,
        pageSize: 20,
        children: []
      }
    }
  },
  created() {
    this.listQuery.id = this.$route.params && this.$route.params.id
    this.getParmList()
  },
  activated() {
    this.getParmList()
  },
  methods: {
    openRestoreOpt(snapid) {
      this.dialogFormVisible = true
      this.restoreOpt.snapid = snapid
      this.restoreOpt.dirCur = '/'
      this.restoreOpt.dist = this.listQuery.path
      this.restoreOpt.include = this.listQuery.path
    },
    restoreFileHandler(data) {
      this.dialogFormVisible = true
      this.restoreOpt.snapid = this.curSnap.short_id
      this.restoreOpt.dirCur = '/'
      this.restoreOpt.dist = data.path
      this.restoreOpt.include = data.path
    },
    openDirSelect() {
      this.dialogDirVisible = true
      this.dirList = []
      this.lsDir(this.restoreOpt.dirCur, true)
    },
    confirmDirSelect(path) {
      if (path) {
        this.restoreOpt.dirCur = path
        this.restoreOpt.dist = path + this.restoreOpt.include
      }
      this.dialogDirVisible = false
    },
    getDirSpea() {
      var dirs = this.restoreOpt.dirCur.split("/")
      dirs.shift()
      var res = []
      var path = ''
      dirs.forEach(n => {
        if (n === '') {
          return
        }
        path = path + '/' + n
        res.push({
          name: n,
          path: path
        })
      })
      res.unshift({
        name: this.$t('msg.root'),
        path: '/'
      })
      return res
    },
    selectDir(path, isdir) {
      if (!isdir) {
        return
      }
      this.restoreOpt.dirCur = path
      this.restoreOpt.dist = path + this.restoreOpt.include
    },
    lsDir(path, isdir) {
      if (!isdir) {
        return
      }
      this.restoreOpt.dirCur = path
      this.restoreOpt.dist = path + this.restoreOpt.include
      var q = {
        path: this.restoreOpt.dirCur
      }
      fetchLs(q).then(res => {
        this.dirList = res.data
      })
    },
    getMoreList() {
      this.listQuery.pageNum++
      this.getList()
    },
    handleSearch() {
      this.listQuery.host = ''
      this.listQuery.path = ''
      this.listQuery.pageNum = 1
      this.list = []
      this.filedata = {
        label: 'root',
        path: '',
        type: 'dir',
        isMore: 3,
        pageNum: 1,
        pageSize: 20,
        children: []
      }
      this.fileSearch = {
        type: 0,
        name: '',
        pageNum: 1,
        pageSize: 20,
      }
      this.snapList = []
      if (this.pathQuery.length === 2) {
        this.listQuery.host = this.pathQuery[0]
        this.listQuery.path = this.pathQuery[1]
        this.getList()
      }
    },
    searchFile() {
      if (this.fileSearch.type === 1) {
        if (this.fileSearch.name === '') {
          this.loadSnapFiles(this.curSnap)
          return
        }
        const q = {
          path: this.fileSearch.name,
          pageNum: this.fileSearch.pageNum,
          pageSize: this.fileSearch.pageSize
        }
        this.filedata.children = []
        this.treeLoading = true
        fetchSearchList(this.listQuery.id, this.curSnap.short_id, q).then(res => {
          const num = res.data.pageNum
          const size = res.data.pageSize
          const nodes = res.data.items.nodes
          nodes.forEach(node => {
            const newChild = {
              pageNum: num,
              pageSize: size,
              name: node.name,
              path: node.path,
              label: node.path,
              type: node.type,
              mode: node.mode,
              isMore: 0,
              permissions: node.permissions,
              ctime: node.ctime,
              gid: node.gid,
              uid: node.uid,
              size: node.size,
              children: []
            }
            this.filedata.children.push(newChild)
          })
        }).finally(() => {
          this.treeLoading = false
        })
      } else {
        this.$notify.error(this.$t('msg.noFunction'))
      }
    },
    restoreSnapHandler() {
      const snapid = this.restoreOpt.snapid
      this.$confirm(this.$t('restore.confirmMessage', {id: snapid}), this.$t('restore.title'), {
        type: 'warning'
      }).then(() => {
        const data = {
          target: this.restoreOpt.dirCur,
          include: this.restoreOpt.include,
          verify: this.restoreOpt.verify
        }
        fetchRestore(this.listQuery.id, snapid, data).then(res => {
          this.dialogDirVisible = false
          this.dialogFormVisible = false
          this.$notify.success({
            title: this.$t('restore.inProgress'),
            message: this.$t('restore.checkTaskMessage'),
            dangerouslyUseHTMLString: true
          })
        }).finally(() => {
          this.dialogDirVisible = false
          this.dialogFormVisible = false
        })
      }).catch(() => {
        this.$notify.info({title: this.$t('common.cancel')})
      })
    },
    loadSnapFiles(snap) {
      this.fileSearch.type = 1
      this.curSnap = snap
      this.filedata.pageNum = 1
      this.filedata.isMore = 3
      this.filedata.path = this.listQuery.path
      this.filedata.children = [{
        pageNum: this.filedata.pageNum,
        pageSize: this.filedata.pageSize,
        name: '',
        path: this.filedata.path,
        label: this.$t('msg.tips.loadMore'),
        isMore: 1,
        type: 'btn',
        mode: 755,
        permissions: '',
        ctime: '',
        gid: '',
        uid: '',
        size: '',
        children: []
      }]
      this.getFiles(this.filedata, {
        parent: {
          data: {
            children: [],
            pageNum: 1
          }
        }
      })
    },
    // 节点点击事件
    moreClick(data, treenode) {
      if (data.type === 'btn') {
        this.getFiles(data, treenode)
      }
    },
    // 展开节点事件
    getFiles(data, treenode) {
      // data.isMore = 0 普通文件，1 加载更多按钮，2 没有更多按钮，3 root 节点
      // data.type = dir 文件夹，file 文件，btn 加载按钮
      // 仅允许root节点点击、文件夹第一次点击、加载更多按钮点击
      if ((data.isMore === 0 && data.type === 'file') || data.isMore === 2) {
        return
      }
      if (data.type === 'dir') {
        //root节点点击、文件夹第一次点击，加载第一页数据，设置没有更多状态
        data.pageNum = 1
      } else {
        //加载更多
        data.pageNum++
      }
      const q = {
        path: data.path,
        pageNum: data.pageNum,
        pageSize: data.pageSize
      }
      this.treeLoading = true
      fetchLsList(this.listQuery.id, this.curSnap.short_id, q).then(res => {
        const num = res.data.pageNum
        const size = res.data.pageSize
        const total = res.data.total
        const nodes = res.data.items.nodes
        nodes.forEach(node => {
          var dirChild = []
          if (node.type === 'dir') {
            dirChild = [{
              pageNum: 1,
              pageSize: size,
              path: node.path,
              name: '',
              label: this.$t('msg.tips.loadMore'),
              isMore: 1,
              type: 'btn',
              mode: node.mode,
              permissions: node.permissions,
              ctime: node.ctime,
              gid: node.gid,
              uid: node.uid,
              size: node.size,
              children: []
            }]
          }
          const newChild = {
            pageNum: 1,
            pageSize: size,
            name: node.name,
            path: node.path,
            label: node.name,
            isMore: 0,
            type: node.type,
            mode: node.mode,
            permissions: node.permissions,
            ctime: node.ctime,
            gid: node.gid,
            uid: node.uid,
            size: node.size,
            children: dirChild
          }
          if (data.isMore === 1) {
            //追加到同级
            const parent = treenode.parent
            if (!parent.data.children) {
              this.filedata.children.push(newChild)
            } else {
              parent.data.children.push(newChild)
            }

          } else {
            //追加到下一级
            data.children.push(newChild)
          }
        })
        if (data.type !== 'file') {
          let child;
          if (data.isMore === 1) {
            const parent = treenode.parent
            if (!parent.data.children) {
              child = this.filedata.children
              this.filedata.children = this.moveMoretoLast(child, num, size, total)
            } else {
              child = parent.data.children
              parent.data.children = this.moveMoretoLast(child, num, size, total)
              if (Number(num) * Number(size) >= total) {
                parent.data.isMore = 0
              }
            }
          } else {
            child = data.children
            data.children = this.moveMoretoLast(child, num, size, total)
            if (Number(num) * Number(size) >= total) {
              data.isMore = 0
            }
          }
        }
      }).finally(() => {
        this.treeLoading = false
      })
    },
    moveMoretoLast(list, num, size, total) {
      let res = []
      let more = null
      list.forEach(l => {
        if (l.isMore === 1) {
          more = l
          if (Number(num) * Number(size) >= total) {
            more.isMore = 2
            more.label = this.$t('msg.tips.noMoreData')
          }
          return
        }
        res.push(l)
      })
      res.push(more)
      return res
    },
    getParmList() {
      fetchParmsList(this.listQuery.id).then(res => {
        this.tagList = res.data.tags
        this.hostList = this.buildCascader(res.data.parms)
      })
    },
    buildCascader(parms) {
      const resp = []
      parms.forEach(res => {
        const chs = []
        res.paths.forEach(path => {
          const children =
            {
              value: path,
              label: path
            }
          chs.push(children)
        })
        const p = {
          value: res.name,
          label: res.name,
          children: chs
        }
        resp.push(p)
      })
      return resp
    },
    getList() {
      this.listLoading = true
      fetchSnapshotsList(this.listQuery).then(response => {
        this.list = this.list.concat(response.data.items)
        this.snapList = this.accordionList(this.list)
        this.noMore = Number(response.data.pageNum) * Number(response.data.pageSize) >= response.data.total
        this.total = response.data.total
      }).finally(() => {
        this.listLoading = false
      })
    },
    accordionList(list) {
      const res = [];
      list.forEach(l => {
        const name = dateFormat(l.time, 'yyyy-MM');
        if (this.listQuery.date) {
          const date = dateFormat(l.time, 'yyyy-MM-dd');
          const seldate = dateFormat(this.listQuery.date, 'yyyy-MM-dd');
          if (date !== seldate) {
            return
          }
          this.activeName = name
        }
        let find = false;
        res.forEach(r => {
          if (r.name === name) {
            r.list.push(l)
            find = true
          }
        })
        if (!find) {
          res.push({
            name: name,
            list: [l]
          })
        }
      })
      return res
    }
  }
}
</script>

<style lang="scss" scoped>
@import "src/styles/variables";

.active {
  background: $light-blue;
  color: $menuHover;
}

.filenode {

}

.filenodes {
  margin-top: 10px;

  .confirmbtn {
    color: $menuHover;
  }
}

.filenode:hover {
  background-color: $light-blue;
  cursor: pointer;
  color: $menuHover;
}

.breadcrumb-item {
  padding: 5px;
}

.breadcrumb-item:hover {
  background-color: $light-blue;
  cursor: pointer;
  color: $menuHover;

  .title {
    color: $menuHover;
  }
}

.input-with-select {
  background-color: #fff;

  .el-select {
    width: 110px;
  }
}

.file-tree {
  margin-top: 10px;
}

.timeline {
  width: 100%;
  cursor: pointer;
  justify-content: space-between;
  flex: 1;
  display: flex;
  align-items: center;

  .snap {
    padding: 10px 0 10px 0;
    width: 100%;
    font-size: 14px;
  }

  .restore_btn {
    padding: 15px;
    height: 100%;
  }

}

.timeline:hover {
  background-color: $hoverGray;
}

.panel-group {
  padding: 10px;

  .card-panel-col {
    background: #fff;
  }
}

.custom-tree-node {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;

  .file-title {
    font-size: 14px;
    padding-right: 8px;
  }
}
</style>
