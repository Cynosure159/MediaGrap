#!/bin/sh
set -eu
# Run in an extracted, checksum-verified FFmpeg 7.1.1 tree.
./configure \
    --prefix=/usr \
    --bindir=/usr/bin \
    --disable-autodetect \
    --disable-debug \
    --disable-doc \
    --disable-everything \
    --disable-avdevice \
    --disable-avfilter \
    --disable-ffmpeg \
    --disable-ffplay \
    --disable-network \
    --disable-swresample \
    --disable-swscale \
    --enable-ffprobe \
    --enable-protocol=file \
    --enable-demuxer=avi,flac,matroska,mp3,mpegts,mov,ogg,wav \
    --enable-parser=aac,ac3,av1,h264,hevc,mpegaudio,mpeg4video,opus,vorbis,vp9 \
    --enable-small \
    --enable-static \
    --disable-shared \
    --extra-cflags="-Os" \
    --extra-ldflags="-static"
mkdir -p /materials/ffmpeg
# Add only linker-map diagnostics; leave all configured compile/link flags intact.
if ! make -j2 V=1 LD="gcc -Wl,-Map=/materials/ffmpeg/link.map" >/materials/ffmpeg/build.log 2>&1; then
    tail -60 /materials/ffmpeg/build.log
    exit 1
fi
make install DESTDIR=/out
cp config.h ffbuild/config.mak ffbuild/config.log /materials/ffmpeg/
# NASM configuration exists only on architectures that use NASM.
if [ -f config.asm ]; then cp config.asm /materials/ffmpeg/; fi
cp COPYING* LICENSE.md /materials/ffmpeg/
/out/usr/bin/ffprobe -L >/materials/ffmpeg/license.txt 2>&1
/out/usr/bin/ffprobe -version >/materials/ffmpeg/version.txt 2>&1
/out/usr/bin/ffprobe -buildconf >/materials/ffmpeg/buildconf.txt 2>&1
grep -q 'LGPL version 2.1 or later' config.h
apk list --installed >/materials/apk-packages.txt
gcc --version >/materials/compiler.txt
ld --version >/materials/linker.txt
# libssp_nonshared is supplied by musl-dev on this toolchain, not assumed GCC.
apk info -W /usr/lib/libssp_nonshared.a >/materials/static-ownership.txt
ar t /usr/lib/libssp_nonshared.a >>/materials/static-ownership.txt
sha256sum /out/usr/bin/ffprobe /etc/ssl/certs/ca-certificates.crt >/materials/runtime-sha256.txt
