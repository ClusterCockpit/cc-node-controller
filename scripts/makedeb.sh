#!/bin/sh

set -eu

BASEDIR="$(pwd)"
WORKSPACE="${BASEDIR}/.dpkgbuild"
DEBIANBINDIR="${WORKSPACE}/DEBIAN"
mkdir -p "${WORKSPACE}" "${DEBIANBINDIR}"
# If this line below fails, please tag your commit. Otherwise we cannot generate
# a valid version number for the DEB.
VERS="$(git describe --tags --abbrev=0 HEAD)"
VERS="${VERS#v}"
ARCH="$(uname -m)"
if [ "${ARCH}" = "x86_64" ]; then
    ARCH=amd64
fi

SIZE_BYTES="$(du -bcs --exclude=.dpkgbuild "${WORKSPACE}"/ | awk '{print $$1}' | head -1 | sed -e 's/^0\+//')"
SIZE="$(awk -v size="${SIZE_BYTES}" 'BEGIN {print (size/1024)+1}' | awk '{print int($0)}')"

CONTROLFILE="${BASEDIR}/scripts/cc-node-controller.deb.control"
sed -e "s+{VERSION}+${VERS}+g" -e "s+{INSTALLED_SIZE}+${SIZE}+g" -e "s+{ARCH}+${ARCH}+g" "${CONTROLFILE}" > "${DEBIANBINDIR}/control"
install -Dpm 755 cc-node-controller "${WORKSPACE}/usr/bin/cc-node-controller"
DEB_FILE="cc-node-controller_${VERS}_${ARCH}.deb"
dpkg-deb --root-owner-group -b "${WORKSPACE}" "${DEB_FILE}"
if [ ! -z "${GITHUB_ACTIONS+x}" ]; then
    echo "DEB=${DEB_FILE}" >> "${GITHUB_OUTPUT}"
fi
rm -r "${WORKSPACE}"
