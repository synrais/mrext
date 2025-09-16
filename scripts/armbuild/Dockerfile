# glibc in this version matches mister
FROM arm32v7/debian:bullseye

# enable bullseye-backports repo manually
RUN echo "deb http://deb.debian.org/debian bullseye-backports main" >> /etc/apt/sources.list && \
    apt-get update -y && apt-get upgrade -y

# make go mod download work
RUN apt-get install -y ca-certificates openssl && \
    openssl s_client -showcerts -connect github.com:443 </dev/null 2>/dev/null | \
      openssl x509 -outform PEM > /usr/local/share/ca-certificates/github.crt && \
    openssl s_client -showcerts -connect proxy.golang.org:443 </dev/null 2>/dev/null | \
      openssl x509 -outform PEM > /usr/local/share/ca-certificates/proxy.golang.crt && \
    update-ca-certificates

# install go and app dependencies
RUN apt-get install -y build-essential git curl wget && \
    apt-get install -y \
        golang-doc/bullseye-backports \
        golang-go/bullseye-backports \
        golang-src/bullseye-backports \
        golang/bullseye-backports && \
    apt-get install -y libsqlite3-dev libncurses-dev linux-source

# install mplayer dependencies
RUN apt-get install -y libasound2-dev libasound2 liblapack-dev liblapack3
RUN apt-get install -y gnutls-bin gnutls-dev libssl-dev openssl libssh2-1-dev libc-ares-dev

# drop permissions on output files
RUN useradd -m -u 1000 build
USER build

# install mage
RUN cd /home/build && \
    git clone https://github.com/magefile/mage --depth 1 && \
    mkdir /home/build/go && \
    cd mage && go run bootstrap.go

ENV PATH="${PATH}:/home/build/go/bin"

WORKDIR /build
RUN git config --global --add safe.directory /build
