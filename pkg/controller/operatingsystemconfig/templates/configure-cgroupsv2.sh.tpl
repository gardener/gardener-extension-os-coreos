#!/bin/bash

KUBELET_CONFIG=/var/lib/kubelet/config/kubelet

if [[ -e /sys/fs/cgroup/cgroup.controllers ]]; then
        echo "CGroups V2 are used!"
        echo "=> Patch kubelet to use systemd as cgroup driver"
        sed -i "s/cgroupDriver: cgroupfs/cgroupDriver: systemd/" "$KUBELET_CONFIG"
else
        echo "No CGroups V2 used by system"
fi
