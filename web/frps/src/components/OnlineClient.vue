<template>
  <div>
    <el-table :data="clients" :default-sort="{prop: 'runid', order: 'ascending'}" style="width: 100%">
      <el-table-column type="expand">
        <template scope="props">
          <el-popover
            ref="popover4"
            placement="right"
            width="600"
  		  style="margin-left:0px"
            trigger="click">
            <my-traffic-chart :proxy_name="props.row.name"></my-traffic-chart>
          </el-popover>
  
          <el-button v-popover:popover4 type="primary" size="small" icon="view" style="margin-bottom:10px">Online Clients Statistics</el-button>
  
          <el-form label-position="left" inline class="demo-table-expand">
            <el-form-item label="RunId">
              <span>{{ props.row.runid }}</span>
            </el-form-item>
            <el-form-item label="ProxyNum">
              <span>{{ props.row.proxy_num }}</span>
            </el-form-item>
            <el-form-item label="ConnNum">
              <span>{{ props.row.conn_num }}</span>
            </el-form-item>
            <el-form-item label="Last Start">
              <span>{{ props.row.last_start_time }}</span>
            </el-form-item>
            <el-form-item label="Last Close">
              <span>{{ props.row.last_close_time }}</span>
            </el-form-item>
        </el-form>
    </template>
    </el-table-column>
    <el-table-column
      label="RunId"
      prop="runid"
      sortable>
    </el-table-column>
    <el-table-column
      label="ProxyNum"
      prop="proxy_num"
      sortable>
    </el-table-column>
    <el-table-column
      label="ConnNum"
      prop="conn_num"
      sortable>
    </el-table-column>
</el-table>
 <pagination :totalPage="parentTotalPage" :currentPage="parentCurrentpage" :changeCallback="fetchData"></pagination> 
</div>
</template>

<script>
  import Humanize from 'humanize-plus';
  import pagination from '../utils/pagination.vue';
  import { Client } from '../utils/client.js';
  
  export default {
    data() {
      return {
        clients: null,
        parentTotalPage: 1,
        parentCurrentpage: 1
      }
    },
    created() {
      this.fetchData(0)
    },
    watch: {
      '$route': 'fetchData'
    },
    components: { pagination },
    methods: {
      fetchData(cPage) {
        if (cPage == 0) {
          fetch('/api/client/online', {credentials: 'include'})
            .then(res => {
              return res.json()
            }).then(json => {
              this.parentTotalPage = json.total_page
              this.parentCurrentPage = 1
              
              fetch('/api/client/online/1', {credentials: 'include'})
              .then(res => {
                return res.json()
              }).then(json => {
                this.clients = new Array()
                for (let clientStats of json.clients) {
                  this.clients.push(new Client(clientStats))
                }
              })
            })
        } else {
          fetch('/api/client/online/'+cPage, {credentials: 'include'})
            .then(res => {
              return res.json()
            }).then(json => {
              this.clients = new Array()
              for (let clientStats of json.clients) {
                this.clients.push(new Client(clientStats))
              }
            })
          }
      } // end fetchData
    } // end method
  } // end default
</script>

<style>
</style>
