#!/bin/bash

CONTAINERD_CONFIG=/etc/containerd/config.toml

ALTERNATE_LOGROTATE_PATH="/usr/bin/logrotate"

# prefer containerd from torcx. Remove this special case once all flatcar versions
# that use torcx have run out of support
CONTAINERD="/usr/bin/containerd"
if [ -x /run/torcx/unpack/docker/bin/containerd ]; then
    CONTAINERD="/run/torcx/unpack/docker/bin/containerd"
fi

# initialize default containerd config if does not exist
if [ ! -s "$CONTAINERD_CONFIG" ]; then
    mkdir -p "$(dirname "$CONTAINERD_CONFIG")"
    ${CONTAINERD} config default > "$CONTAINERD_CONFIG"
    chmod 0644 "$CONTAINERD_CONFIG"
fi

# if cgroups v2 are used, patch containerd configuration to use systemd cgroup driver
if [[ -e /sys/fs/cgroup/cgroup.controllers ]]; then
    sed -i "s/SystemdCgroup *= *false/SystemdCgroup = true/" "$CONTAINERD_CONFIG"
fi

# Remove this block once all flatcar versions that use torcx have run out of support
# provide kubelet with access to the containerd binaries in /run/torcx/unpack/docker/bin
if [ ! -s /etc/systemd/system/kubelet.service.d/environment.conf ]; then
    mkdir -p /etc/systemd/system/kubelet.service.d/
    cat <<EOF | tee /etc/systemd/system/kubelet.service.d/environment.conf
[Service]
Environment="PATH=/run/torcx/unpack/docker/bin:$PATH"
EOF
    chmod 0644 /etc/systemd/system/kubelet.service.d/environment.conf
    systemctl daemon-reload
fi

# some flatcar versions have logrotate at /usr/bin instead of /usr/sbin
if [ -f "$ALTERNATE_LOGROTATE_PATH" ]; then
    sed -i "s;/usr/sbin/logrotate;$ALTERNATE_LOGROTATE_PATH;" /etc/systemd/system/containerd-logrotate.service
    systemctl daemon-reload
fi
