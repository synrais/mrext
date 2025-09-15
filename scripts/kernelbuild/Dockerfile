FROM ubuntu:kinetic

# Point apt sources to old-releases since kinetic is EOL
RUN sed -i 's|http://archive.ubuntu.com/ubuntu|http://old-releases.ubuntu.com/ubuntu|g' /etc/apt/sources.list && \
    sed -i 's|http://security.ubuntu.com/ubuntu|http://old-releases.ubuntu.com/ubuntu|g' /etc/apt/sources.list

# Now update and upgrade
RUN apt-get update -y && apt-get upgrade -y

# Install all required dependencies + Go
RUN apt-get install -y build-essential git libncurses-dev flex bison openssl libssl-dev dkms libelf-dev \
    libudev-dev libpci-dev libiberty-dev autoconf liblz4-tool bc curl gcc libssl-dev libncurses5-dev \
    lzop make u-boot-tools libgmp3-dev libmpc-dev wget golang

WORKDIR /opt

# Install ARM cross-toolchain (Linaro)
RUN wget -c https://releases.linaro.org/components/toolchain/binaries/6.5-2018.12/arm-linux-gnueabihf/gcc-linaro-6.5.0-2018.12-x86_64_arm-linux-gnueabihf.tar.xz && \
    tar -xJf gcc-linaro-6.5.0-2018.12-x86_64_arm-linux-gnueabihf.tar.xz

# Create non-root user for builds
RUN useradd -m -u 1000 build
USER build

# Environment for cross-compilation
ENV CROSS_COMPILE=/opt/gcc-linaro-6.5.0-2018.12-x86_64_arm-linux-gnueabihf/bin/arm-linux-gnueabihf-
ENV ARCH=arm
ENV LOCALVERSION=-MiSTer

WORKDIR /build
