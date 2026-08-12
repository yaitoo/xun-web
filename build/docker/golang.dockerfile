FROM debian:12-slim AS yaitoo-golang

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

# Download tailwindcss to a local directory, then expose it via the system PATH.
RUN mkdir -p /opt/bin && \
    curl -fsSLk -o /opt/bin/tailwindcss \
        https://yaitoo.cn/tailwindcss && \
    chmod +x /opt/bin/tailwindcss && \
    ln -sf /opt/bin/tailwindcss /usr/local/bin/tailwindcss

# Download esbuild to a local directory, then expose it via the system PATH.
RUN curl -fsSL -o /opt/bin/esbuild \
        https://cdn.jsdelivr.net/npm/@esbuild/linux-x64@0.28.0/bin/esbuild && \
    chmod +x /opt/bin/esbuild && \
    ln -sf /opt/bin/esbuild /usr/local/bin/esbuild

# Switch /bin/sh to bash so subsequent RUN scripts (incl. Go's post-install)
# get a real shell with $(...) and arrays. Done before Go install.
RUN rm /bin/sh && ln -s /bin/bash /bin/sh

# Go is installed last so version bumps only invalidate this final layer —
# the apt / tailwindcss / esbuild layers above stay cached.
RUN curl -L -o go.linux-amd64.tar.gz https://go.dev/dl/go1.26.4.linux-amd64.tar.gz
RUN tar -C /usr/local -vxzf go.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

ENV GOPROXY=https://mirrors.cloud.tencent.com/go/,direct