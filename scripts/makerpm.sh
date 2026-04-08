#!/bin/sh

set -eu

BASEDIR="$(pwd)"
SPECFILE="${BASEDIR}/scripts/cc-node-controller.spec"

# Setup RPM build tree
eval $(rpm --eval "ARCH='%{_arch}' RPMDIR='%{_rpmdir}' SOURCEDIR='%{_sourcedir}' SPECDIR='%{_specdir}' SRPMDIR='%{_srcrpmdir}' BUILDDIR='%{_builddir}'")
mkdir --parents --verbose "${RPMDIR}" "${SOURCEDIR}" "${SPECDIR}" "${SRPMDIR}" "${BUILDDIR}"

# Create source tarball
COMMITISH="HEAD"
VERS="$(git describe --tags "${COMMITISH}")"
VERS="${VERS#v}"
VERS=$(echo "${VERS}" | sed -e s+'-'+'_'+g)
eval $(rpmspec --query --queryformat "NAME='%{name}' VERSION='%{version}' RELEASE='%{release}' NVR='%{NVR}' NVRA='%{NVRA}'" --define="VERS ${VERS}" "${SPECFILE}")
PREFIX="${NAME}-${VERSION}"
FORMAT="tar.gz"
SRCFILE="${SOURCEDIR}/${PREFIX}.${FORMAT}"
git archive --verbose --format "${FORMAT}" --prefix="${PREFIX}/" --output="${SRCFILE}" "${COMMITISH}"
# Build RPM and SRPM
rpmbuild -ba --define="VERS ${VERS}" --rmsource --clean "${SPECFILE}"
# Report RPMs and SRPMs when in GitHub Workflow
if [ ! -z "${GITHUB_ACTIONS+x}" ]; then
     RPMFILE="${RPMDIR}/${ARCH}/${NVRA}.rpm"
     SRPMFILE="${SRPMDIR}/${NVR}.src.rpm"
     echo "SRPM=${SRPMFILE}" >> "${GITHUB_OUTPUT}"
     echo "RPM=${RPMFILE}" >> "${GITHUB_OUTPUT}"
fi
