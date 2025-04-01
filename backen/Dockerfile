FROM centos:7

ENV GOVERSION=1.10.3
RUN \ 
  cd /usr/local && \  
  yum install -y wget git gcc gdb make && \
  wget https://dl.google.com/go/go${GOVERSION}.linux-amd64.tar.gz && \
  tar -xzvf go${GOVERSION}.linux-amd64.tar.gz && \
  rm -f go${GOVERSION}.linux-amd64.tar.gz

ENV GOROOT=/usr/local/go
ENV PATH=$PATH:/usr/local/go/bin:/root/go/bin
RUN go get -u github.com/go-delve/delve/cmd/dlv && \
    go get -u github.com/cweill/gotests/...

ENV GOPATH=/root/go
RUN mv /root/go/bin/dlv /usr/local/go/bin/
WORKDIR /root/go