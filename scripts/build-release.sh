#!/usr/bin/env bash
# 构建发布产物：三平台 × 双架构 × 全部工具（零第三方依赖，无 cgo 依赖 → 直接交叉编译）。
#
# 用法：scripts/build-release.sh [版本号] [输出目录]
#   VERSION 文件 / git describe 是默认版本来源；版本经 -ldflags -X main.version 注入二进制。
# 环境变量：QUARK_TARGETS 覆盖目标矩阵（默认 linux/darwin/windows × amd64/arm64）
set -euo pipefail

cd "$(dirname "$0")/.."
VERSION="${1:-$(cat VERSION 2>/dev/null || echo 0.0.0-dev)}"
OUT="${2:-dist}"
TARGETS="${QUARK_TARGETS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}"

# 工具清单：名字 → 构建路径（主模块 / 编译器子模块）
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
# 归一为绝对路径：编译器子模块用 (cd compiler && go build -o ...) 构建，
# 相对路径会被解释成 repo/tmp/... 之类的错误位置（绝对路径 + ../ 拼接的坑）。
OUT="$(cd "$OUT" && pwd)"

echo "→ 版本 ${VERSION}；目标：${TARGETS}"
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

# 清单：文件、大小、sha256（发布校验用）
MANIFEST="$OUT/MANIFEST-${VERSION}.txt"
{
  echo "# QuarkLang ${VERSION} 发布产物"
  echo "# 生成时间: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  echo "# 构建: CGO_ENABLED=0（纯 Go，无系统依赖）"
  printf '%-44s %12s  %s\n' "文件" "字节" "sha256"
  for f in "$OUT"/*; do
    [ "$(basename "$f")" = "$(basename "$MANIFEST")" ] && continue
    printf '%-44s %12s  %s\n' "$(basename "$f")" "$(stat -c%s "$f")" "$(sha256sum "$f" | cut -d' ' -f1)"
  done
} > "$MANIFEST"

echo "✓ 产物目录: $OUT（$(ls "$OUT" | grep -vc MANIFEST) 个二进制）"
echo "✓ 清单: $MANIFEST"
