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
  
          <el-button v-popover:popover4 type="primary" size="small" icon="view" style="margin-bottom:10px">Offline Clients Statistics</el-button>
  
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
</div>
</template>

<script>
  import Humanize from 'humanize-plus';
  import {
    Client
  } from '../utils/client.js'
  export default {
    data() {
      return {
        clients: null,
        vhost_http_port: "",
        subdomain_host: ""
      }
    },
    created() {
      this.fetchData()
    },
    watch: {
      '$route': 'fetchData'
    },
    methods: {
      fetchData() {
        fetch('/api/client/offline', {credentials: 'include'})
          .then(res => {
            return res.json()
          }).then(json => {
            this.clients = new Array()
            for (let clientStats of json.clients) {
              this.clients.push(new Client(clientStats))
            }
          })
      }
    },
    components: {
        'my-traffic-chart': Traffic
    }
  }
</script>

<style>
</style>
