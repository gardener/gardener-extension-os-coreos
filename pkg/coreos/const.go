package coreos

const ContainerDUnitDropInContent = `[Service]
ExecStart=
ExecStart=/run/torcx/bin/containerd --config /etc/containerd/config.toml
Restart=on-failure`

const DockerUnitDropInContent = `[Service]
ExecStart=
ExecStart=/run/torcx/bin/dockerd --host=fd:// --containerd=/run/containerd/containerd.sock --selinux-enabled=true`

