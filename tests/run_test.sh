#!/bin/bash

./../bin/xfrps -c ./conf/auto_test_frps.ini &
sleep 1
./../bin/xfrpc -c ./conf/auto_test_frpc.ini &

# wait until proxies are connected
sleep 2
