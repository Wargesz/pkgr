FROM ghcr.io/void-linux/void-glibc-full:20260901r1

WORKDIR /opt/pkgr/

ENV HOST="192.168.0.15"
ENV PREFIX="/.pkgr/bin/"
ENV NPROC=1
RUN mkdir -p /etc/xbps.d
RUN cp /usr/share/xbps.d/*-repository-*.conf /etc/xbps.d/
RUN sed -i 's|https://repo-default.voidlinux.org|https://repo-de.voidlinux.org/|g' /etc/xbps.d/*-repository-*.conf
RUN xbps-install -Syu xbps
RUN xbps-install -Syu wget tar curl zip unzip file make gcc
RUN mkdir build
COPY builder .

CMD ["./builder"]
