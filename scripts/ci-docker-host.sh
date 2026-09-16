#!/usr/bin/env bash
# Prints the DOCKER_HOST that a Forgejo Actions job should use.
#
# Jobs run as containers on a Docker-in-Docker daemon that mounts no socket into
# them, so docker is unusable until DOCKER_HOST points back across the bridge at
# that daemon. The daemon listens on 2375 at the container's default gateway.
#
# The gateway is read from /proc/net/route rather than `ip route`: the runner image
# ships no iproute2. The field is a little-endian hex address, hence the reversed
# byte order below.
set -euo pipefail

hex="$(awk '$2 == "00000000" { print $3; exit }' /proc/net/route)"
if [ ${#hex} -ne 8 ]; then
	echo "could not read a default gateway from /proc/net/route" >&2
	exit 1
fi

printf 'tcp://%d.%d.%d.%d:2375\n' \
	"$((0x${hex:6:2}))" "$((0x${hex:4:2}))" "$((0x${hex:2:2}))" "$((0x${hex:0:2}))"
