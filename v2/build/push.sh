#!/usr/bin/env bash
set -Eeu -o pipefail

# Parse explicit TAG option.
usage() { echo "Usage: $0 [ -t image-tag ]" 1>&2; exit 1; }

TAG="latest"

while getopts ":t:" opt; do
  case "${opt}" in
    t)   TAG=${OPTARG} ;;
    \? ) usage ;;
  esac
done

if [[ "$TAG" = "latest" ]]; then
  docker push liveramp/gazette-vendored:latest
  docker push liveramp/gazette-examples:latest
  docker push liveramp/gazette:latest
else 
  docker tag liveramp/gazette-examples:latest liveramp/gazette-examples:${TAG}
  docker tag liveramp/gazette:latest          liveramp/gazette:${TAG}

  docker push liveramp/gazette-examples:${TAG}
  docker push liveramp/gazette:${TAG}
fi
