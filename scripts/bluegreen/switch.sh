#!/bin/sh
# Points Nginx at the other copy of the web app, then reloads Nginx gracefully: the old workers
# finish the requests they hold while new workers send to the new copy.
#
#   docker compose -f scripts/bluegreen/compose.yaml exec nginx sh /switch.sh green
set -eu

color="${1:-}"
case "$color" in
  blue | green) ;;
  *) echo "usage: switch.sh blue|green" >&2; exit 2 ;;
esac
target="web-$color"

# Like the warm-up before a slot swap: refuse to send traffic to a copy that does not answer.
if ! wget -q -O /dev/null "http://$target:3000/"; then
  echo "$target does not answer; still serving the current copy" >&2
  exit 1
fi

echo "set \$active $target;" > /etc/nginx/active.conf
nginx -t -q
nginx -s reload
echo "now serving $target"
