class Client {
    constructor(clientStats) {
        this.runid      = clientStats.runid
        this.proxy_num  = clientStats.proxy_num
        this.conn_num   = clientStats.conn_num
        this.last_start_time = clientStats.last_start_time
        this.last_close_time = clientStats.last_close_time
    }
}
