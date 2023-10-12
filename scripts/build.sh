#!/usr/bin/env bash

set -e

if [[ "$OUT" == "" ]]; then
  OUT="$PWD/result"
fi

ref_signer="$PWD/external/c-reference-signer"

mkdir -p "$OUT"/{headers,bin}
rm -f "$OUT"/libmina_signer.so # Otherwise re-building without clean causes permissions issue
if [[ "$LIB_MINA_SIGNER" == "" ]]; then
  # No nix
  cp -R "$ref_signer" "$OUT"
  make -C "$OUT/c-reference-signer" clean libmina_signer.so
  cp "$OUT/c-reference-signer/libmina_signer.so" "$OUT"
else
  cp "$LIB_MINA_SIGNER" "$OUT"/libmina_signer.so
fi
cp "$ref_signer"/*.h "$OUT/headers"

case "$1" in
  test)
    cd src/delegation_backend
    LD_LIBRARY_PATH="$OUT" $GO test
    ;;
  docker)
    tag=delegation-backend
    docker build -t "$tag" -f dockerfiles/Dockerfile-delegation-backend .
    ;;
  "")
    cd src/cmd/delegation_backend
    $GO build -o "$OUT/bin/delegation_backend"
    echo "to run use cmd: LD_LIBRARY_PATH=result ./result/bin/delegation_backend"
    ;;
  *)
    echo "unknown command $1"
    exit 2
    ;;
esac
