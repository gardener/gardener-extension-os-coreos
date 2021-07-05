#!/bin/bash

# initiliaze default containerd config if does not exist
if [ ! -s /etc/containerd/config.toml ]; then
    mkdir -p /etc/containerd/
    /run/torcx/unpack/docker/bin/containerd config default > /etc/containerd/config.toml
    chmod 0644 /etc/containerd/config.toml
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
