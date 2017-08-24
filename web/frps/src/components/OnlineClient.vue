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
 <h1>pager used for event</h1>
  <pager
  mode="event"
  :total-page="totalPage"
  :init-page="eventPage"
  @go-page="goPage"></pager>
</div>
</template>

<script>
  import Humanize from 'humanize-plus';
  import { Client } from '../utils/client.js';
  import { pager } from '../utils/vue-pager.vue';
  export default {
    data() {
      return {
        clients: null,
        eventPage: 1,
        queryPage: 1,
        paramsPage: 1,
        totalPage: 10
      }
    },
    created() {
      this.fetchData()
    },
    watch: {
      '$route': 'fetchData'
    },
    components: {
        'pager': pager
    },
    route: {
        data ({to: {query, params}}) {
            if(params.page) {
                this.paramsPage = parseInt(params.page) || 1
            } else {
                this.paramsPage = 1
            }
            if(query.page) {
                this.queryPage = parseInt(query.page) || 1
            } else {
                this.queryPage = 1
            }
        }
    },
    methods: {
      fetchData() {
        fetch('/api/client/online', {credentials: 'include'})
          .then(res => {
            return res.json()
          }).then(json => {
            this.clients = new Array()
            for (let clientStats of json.clients) {
              this.clients.push(new Client(clientStats))
            }
          })
      }, // end fetchData
      goPage (data) {
          this.eventPage = data.page
      } // end goPage
    } // end method
  } // end default
</script>

<style>
</style>
