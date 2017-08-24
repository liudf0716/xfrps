import Vue from 'vue'
import Router from 'vue-router'
import Overview from '../components/Overview.vue'
import ProxiesTcp from '../components/ProxiesTcp.vue'
import ProxiesUdp from '../components/ProxiesUdp.vue'
import ProxiesFtp from '../components/ProxiesFtp.vue'
import ProxiesHttp from '../components/ProxiesHttp.vue'
import ProxiesHttps from '../components/ProxiesHttps.vue'
import OnlineClient from '../components/OnlineClient.vue'
import OfflineClient from '../components/OfflineClient.vue'
import Search from '../components/Search.vue'

Vue.use(Router)

export default new Router({
    routes: [{
        path: '/',
        name: 'Overview',
        component: Overview
    }, {
        path: '/client/offline',
        name: 'OfflineClient',
        component: OfflineClient
    }, {
        path: '/client/online',
        name: 'OnlineClient',
        component: OnlineClient
    }, {
        path: '/proxies/tcp',
        name: 'ProxiesTcp',
        component: ProxiesTcp
    }, {
        path: '/proxies/udp',
        name: 'ProxiesUdp',
        component: ProxiesUdp
    }, {
        path: '/proxies/ftp',
        name: 'ProxiesFtp',
        component: ProxiesFtp
    }, {
        path: '/proxies/http',
        name: 'ProxiesHttp',
        component: ProxiesHttp
    }, {
        path: '/proxies/https',
        name: 'ProxiesHttps',
        component: ProxiesHttps
    }, {
        path:   '/search',
        name:   'Search'
        component:  Search
    }]
})
