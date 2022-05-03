#!/bin/bash

CONTAINERD_CONFIG=/etc/containerd/config.toml

# initialize default containerd config if does not exist
if [ ! -s "$CONTAINERD_CONFIG" ]; then
    mkdir -p /etc/containerd/
    /run/torcx/unpack/docker/bin/containerd config default > "$CONTAINERD_CONFIG"
    chmod 0644 "$CONTAINERD_CONFIG"
fi

# if cgroups v2 are used, patch containerd configuration to use systemd cgroup driver
if [[ -e /sys/fs/cgroup/cgroup.controllers ]]; then
    sed -i "s/SystemdCgroup *= *false/SystemdCgroup = true/" "$CONTAINERD_CONFIG"
fi

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
