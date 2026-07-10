FROM debian:12-slim AS yaitoo-debian

RUN apt-get update -y && \
    apt-get install -y --no-install-recommends ca-certificates

RUN rm -rf /etc/apt/sources.list.d/*

RUN echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm main contrib non-free non-free-firmware" > /etc/apt/sources.list && \
    echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian-security bookworm-security main contrib non-free non-free-firmware" >> /etc/apt/sources.list && \
    echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-updates main contrib non-free non-free-firmware" >> /etc/apt/sources.list && \
    echo "deb https://mirrors.tuna.tsinghua.edu.cn/debian/ bookworm-backports main contrib non-free non-free-firmware" >> /etc/apt/sources.list

RUN apt update -y

RUN apt-get install curl make -y
RUN apt-get install gcc -y
RUN apt-get install git -y
RUN apt-get install g++ -y
RUN apt-get install build-essential -y

RUN apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN curl -L -o go.linux-amd64.tar.gz https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
RUN tar -C /usr/local -vxzf go.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct



RUN rm /bin/sh && ln -s /bin/bash /bin/sh