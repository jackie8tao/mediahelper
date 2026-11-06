#!/bin/bash
# ==============================================================================
# Script: build_linux_pkg.sh
# Purpose: Build standalone Linux tar.gz release bundles and .deb packages
# Supported Architectures: amd64, arm64
# ==============================================================================

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
VERSION="${VERSION:-$(git -C "${ROOT_DIR}" describe --tags --always 2>/dev/null || echo "v1.0.0")}"
# Strip leading 'v' if present for deb packaging
PKG_VERSION="${VERSION#v}"

ARCH="${1:-amd64}" # amd64, arm64, or all

build_arch() {
    local target_arch="$1"
    local deb_arch="$target_arch"
    if [ "$deb_arch" = "x86_64" ]; then
        deb_arch="amd64"
    fi

    echo "=================================================================="
    echo " Building Linux package for: ${target_arch} (Version: ${VERSION})"
    echo "=================================================================="

    local pkg_name="mediahelper-linux-${target_arch}-${VERSION}"
    local staging_dir="${DIST_DIR}/staging/${pkg_name}"
    local deb_staging_dir="${DIST_DIR}/deb_staging/mediahelper_${PKG_VERSION}_${deb_arch}"

    rm -rf "${staging_dir}" "${deb_staging_dir}"
    mkdir -p "${staging_dir}" "${DIST_DIR}"

    # 1. Compile Linux binary statically without CGO
    echo "--> Compiling statically linked binary (GOOS=linux GOARCH=${target_arch})..."
    CGO_ENABLED=0 GOOS=linux GOARCH="${target_arch}" go build \
        -ldflags="-s -w -extldflags '-static'" \
        -o "${staging_dir}/mediahelper" \
        "${ROOT_DIR}/cmd/mediahelper"

    # 2. Stage installation assets for tar.gz bundle
    echo "--> Staging deployment scripts and systemd files..."
    cp "${ROOT_DIR}/scripts/systemd/mediahelper.service" "${staging_dir}/"
    cp "${ROOT_DIR}/scripts/package/config.linux.json" "${staging_dir}/config.json"
    cp "${ROOT_DIR}/scripts/package/install.sh" "${staging_dir}/"
    cp "${ROOT_DIR}/scripts/package/uninstall.sh" "${staging_dir}/"
    chmod +x "${staging_dir}/install.sh" "${staging_dir}/uninstall.sh" "${staging_dir}/mediahelper"

    # 3. Create standalone tar.gz release archive
    local tar_file="${DIST_DIR}/${pkg_name}.tar.gz"
    echo "--> Packaging tar.gz release archive to ${tar_file}..."
    tar -czf "${tar_file}" -C "${DIST_DIR}/staging" "${pkg_name}"
    echo "    Tar archive generated: ${tar_file}"

    # 4. Prepare Debian (.deb) package structure
    echo "--> Preparing Debian package hierarchy..."
    mkdir -p "${deb_staging_dir}/DEBIAN"
    mkdir -p "${deb_staging_dir}/usr/local/bin"
    mkdir -p "${deb_staging_dir}/etc/mediahelper"
    mkdir -p "${deb_staging_dir}/var/lib/mediahelper/data"
    mkdir -p "${deb_staging_dir}/lib/systemd/system"

    # Copy binary and configuration
    install -m 0755 "${staging_dir}/mediahelper" "${deb_staging_dir}/usr/local/bin/mediahelper"
    install -m 0644 "${staging_dir}/config.json" "${deb_staging_dir}/etc/mediahelper/config.json"
    install -m 0644 "${staging_dir}/mediahelper.service" "${deb_staging_dir}/lib/systemd/system/mediahelper.service"

    # Create DEBIAN/control file
    cat <<EOF > "${deb_staging_dir}/DEBIAN/control"
Package: mediahelper
Version: ${PKG_VERSION}
Section: utils
Priority: optional
Architecture: ${deb_arch}
Maintainer: Jackie Tao <jackie8tao@outlook.com>
Description: MediaHelper automated media management pipeline service
 MediaHelper automates media file parsing, scraping, organization, and downloading
 workflows for local media libraries.
EOF

    # Create DEBIAN/postinst script
    cat <<'EOF' > "${deb_staging_dir}/DEBIAN/postinst"
#!/bin/sh
set -e

mkdir -p /var/lib/mediahelper/data
chmod 0755 /var/lib/mediahelper/data

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload || true
    systemctl enable mediahelper.service || true
fi

echo "MediaHelper installed successfully. Start it with: sudo systemctl start mediahelper"
exit 0
EOF
    chmod 0755 "${deb_staging_dir}/DEBIAN/postinst"

    # Create DEBIAN/prerm script
    cat <<'EOF' > "${deb_staging_dir}/DEBIAN/prerm"
#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
    systemctl stop mediahelper.service || true
    systemctl disable mediahelper.service || true
fi

exit 0
EOF
    chmod 0755 "${deb_staging_dir}/DEBIAN/prerm"

    # 5. Build .deb package using dpkg-deb or fallback to standard ar + tar
    local deb_output="${DIST_DIR}/mediahelper_${PKG_VERSION}_${deb_arch}.deb"
    if command -v dpkg-deb >/dev/null 2>&1; then
        echo "--> Building .deb with host dpkg-deb..."
        dpkg-deb --build --root-owner-group "${deb_staging_dir}" "${deb_output}"
        echo "    Debian package generated: ${deb_output}"
    elif command -v ar >/dev/null 2>&1; then
        echo "--> Host dpkg-deb not found. Assembling standard .deb package using ar and tar..."
        local deb_work_dir="${DIST_DIR}/deb_work_${deb_arch}"
        rm -rf "${deb_work_dir}"
        mkdir -p "${deb_work_dir}"

        # 5.1 debian-binary
        echo "2.0" > "${deb_work_dir}/debian-binary"

        # 5.2 control.tar.gz
        (cd "${deb_staging_dir}/DEBIAN" && tar -czf "${deb_work_dir}/control.tar.gz" ./*)

        # 5.3 data.tar.gz (excluding DEBIAN folder)
        (cd "${deb_staging_dir}" && tar --exclude='./DEBIAN' -czf "${deb_work_dir}/data.tar.gz" ./*)

        # 5.4 Assemble deb using ar
        (cd "${deb_work_dir}" && ar -cr "${deb_output}" debian-binary control.tar.gz data.tar.gz)
        rm -rf "${deb_work_dir}"
        echo "    Debian package generated: ${deb_output}"
    else
        echo "[WARN] Neither dpkg-deb nor ar found. Skipping .deb creation."
    fi

    echo ""
    echo "Finished packaging for ${target_arch}."
}

# Main execution
if [ "${ARCH}" = "all" ]; then
    build_arch "amd64"
    build_arch "arm64"
else
    build_arch "${ARCH}"
fi

# Cleanup staging directories
rm -rf "${DIST_DIR}/staging" "${DIST_DIR}/deb_staging"

echo "=================================================================="
echo " All packages ready in: ${DIST_DIR}"
echo "=================================================================="
ls -lh "${DIST_DIR}"

