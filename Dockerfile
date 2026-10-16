# Built by GoReleaser (dockers_v2): the binary for each platform is copied
# into the build context under $TARGETPLATFORM. There is nothing else in the
# image: no shell, no libc, no package manager.
FROM scratch
ARG TARGETPLATFORM
COPY LICENSE /LICENSE
COPY $TARGETPLATFORM/gs1 /gs1
ENTRYPOINT ["/gs1"]
CMD ["help"]
