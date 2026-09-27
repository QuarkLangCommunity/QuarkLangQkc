#!/usr/bin/env bash
# Build release artifacts: three platforms x two architectures x all tools (zero third-party deps, no cgo -> straight cross-compilation).
#
# Usage: scripts/build-release.sh [version] [output dir]
#   The VERSION file / git describe are the default version sources; the version is injected into the binaries with -ldflags -X main.version.
# Environment: QUARK_TARGETS overrides the target matrix (default linux/darwin/windows x amd64/arm64)
set -euo pipefail

cd "$(dirname "$0")/.."
VERSION="${1:-$(cat VERSION 2>/dev/null || echo 0.0.0-dev)}"
OUT="${2:-dist}"
TARGETS="${QUARK_TARGETS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}"

# Tool list: name -> build path (main module / compiler submodule)
TOOLS=(
  "quark:."
  "qkcheck:./cmd/qkcheck"
  "qkdoc:./cmd/qkdoc"
  "qkrepl:./cmd/qkrepl"
  "qklsp:./cmd/qklsp"
)
COMPILER_TOOLS=("qkc:.")

LDFLAGS="-s -w -X main.version=${VERSION}"

rm -rf "$OUT"
mkdir -p "$OUT"
# Normalize to an absolute path: the compiler submodule is built with (cd compiler && go build -o ...),
# where a relative path would be resolved to the wrong place such as repo/tmp/... (the absolute path + ../ pitfall).
OUT="$(cd "$OUT" && pwd)"

echo "→ version ${VERSION}; targets: ${TARGETS}"
for target in $TARGETS; do
  GOOS="${target%%/*}"
  GOARCH="${target##*/}"
  ext=""
  [ "$GOOS" = "windows" ] && ext=".exe"
  for entry in "${TOOLS[@]}"; do
    name="${entry%%:*}"
    pkg="${entry##*:}"
    out="$OUT/${name}-${VERSION}-${GOOS}-${GOARCH}${ext}"
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath -ldflags "$LDFLAGS" -o "$out" "$pkg"
    printf '  ✓ %s\n' "$(basename "$out")"
  done
  for entry in "${COMPILER_TOOLS[@]}"; do
    name="${entry%%:*}"
    pkg="${entry##*:}"
    out="$OUT/${name}-${VERSION}-${GOOS}-${GOARCH}${ext}"
    (cd compiler && CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath -ldflags "$LDFLAGS" -o "$out" "$pkg")
    printf '  ✓ %s\n' "$(basename "$out")"
  done
done

# Manifest: file, size, sha256 (used to verify a release)
MANIFEST="$OUT/MANIFEST-${VERSION}.txt"
{
  echo "# QuarkLang ${VERSION} release artifacts"
  echo "# Generated: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  echo "# Build: CGO_ENABLED=0 (pure Go, no system dependencies)"
  printf '%-44s %12s  %s\n' "file" "bytes" "sha256"
  for f in "$OUT"/*; do
    [ "$(basename "$f")" = "$(basename "$MANIFEST")" ] && continue
    printf '%-44s %12s  %s\n' "$(basename "$f")" "$(stat -c%s "$f")" "$(sha256sum "$f" | cut -d' ' -f1)"
  done
} > "$MANIFEST"

echo "✓ artifact directory: $OUT ($(ls "$OUT" | grep -vc MANIFEST) binaries)"
echo "✓ manifest: $MANIFEST"
