#!/bin/sh
set -e
H=${1:?usage: ./build.sh https://files.example.com}
[ -f .secret ] || (umask 077 && openssl rand -hex 32 > .secret)
[ -f .pass ] || (umask 077 && printf %s "${PASS:-$(openssl rand -hex 12)}" > .pass)
docker build --no-cache --build-arg SECRET="$(cat .secret)" --build-arg HOST="$H" --build-arg PASS="$(cat .pass)" -t files .
C=$(docker create files)
docker cp "$C":/upload ./upload
docker rm "$C" >/dev/null
chmod +x upload
mkdir -p data
echo "upload binary: ./upload <file>"
echo "curl: curl -T <file> -H \"X-Secret: \$(cat .secret)\" $H/u/"
