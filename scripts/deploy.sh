#!/bin/sh
# Copies a git ref (default: main) to the server and restarts it there.
#   VUC_HOST=root@1.2.3.4 ./scripts/deploy.sh [ref]
# The server keeps /opt/vuc/server.env and /opt/vuc/.secrets; this script
# does not touch them.
set -eu
cd "$(dirname "$0")/.."
ref=${1:-main}
host=${VUC_HOST:?Set VUC_HOST, e.g. root@1.2.3.4}
key=${VUC_SSH_KEY:-$HOME/.ssh/vuc_vps_ed25519}
commit=$(git rev-parse --short "$ref")
echo "Deploying $ref ($commit) to $host"
git archive --format=tar "$ref" | ssh -i "$key" "$host" "
	set -e
	rm -rf /opt/vuc/project.new && mkdir -p /opt/vuc/project.new
	tar -x -C /opt/vuc/project.new
	echo $commit > /opt/vuc/project.new/REVISION
	rm -rf /opt/vuc/project.old
	if [ -d /opt/vuc/project ]; then mv /opt/vuc/project /opt/vuc/project.old; fi
	mv /opt/vuc/project.new /opt/vuc/project
	cd /opt/vuc/project
	docker compose -f docker-compose.prod.yaml --env-file ../server.env up -d --build
	docker image prune -f >/dev/null
"
