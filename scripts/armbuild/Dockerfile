# Use a base that exists
FROM arm32v7/debian:bullseye

# Add backports repo manually
RUN echo "deb http://deb.debian.org/debian bullseye-backports main" > /etc/apt/sources.list.d/backports.list

RUN apt-get update -y && apt-get upgrade -y

# CA certs for Go modules
RUN apt-get install -y ca-certificates openssl && update-ca-certificates

# Build + Go from backports
RUN apt-get install -y --no-install-recommends \
    build-essential git curl wget pkg-config \
    golang-go/bullseye-backports golang-src/bullseye-backports

# Libraries needed for mrext
RUN apt-get install -y --no-install-recommends \
    libsqlite3-dev libncurses-dev libasound2-dev liblapack-dev \
    gnutls-bin libssl-dev libssh2-1-dev libc-ares-dev

# Add non-root user
RUN useradd -m -u 1000 build
USER build

WORKDIR /build
