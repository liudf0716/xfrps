FROM golang:1.8

COPY . /go/src/github.com/KunTengRom/xfrps

RUN cd /go/src/github.com/KunTengRom/xfrps \
 && make \
 && mv bin/xfrpc /xfrpc \
 && mv bin/xfrps /xfrps \
 && mv conf/frpc_min.ini /frpc.ini \
 && mv conf/frps_min.ini /frps.ini \
 && make clean

WORKDIR /

EXPOSE 80 443 6000 7000 7500

ENTRYPOINT ["/xfrps"]
