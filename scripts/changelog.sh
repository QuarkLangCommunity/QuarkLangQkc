#!/usr/bin/env bash
# 从 git 历史生成变更日志（按类型分组），用于 GitHub Release 说明与 CHANGELOG.md。
#
# 用法：scripts/changelog.sh [版本号] [起始 ref]
#   scripts/changelog.sh 0.3.0            # 自上一个 tag 以来 → 「## 0.3.0 - 日期」
#   scripts/changelog.sh 0.3.0 v0.2.0     # 指定起始 ref
#   scripts/changelog.sh --all            # 全部历史（按 tag 分段）
set -euo pipefail

cd "$(dirname "$0")/.."

# emit_commits <range>：按关键词把提交分到各组（仓库提交信息为中文自由格式）。
emit_commits() {
  local range="$1"
  local log
  log="$(git log --no-merges --pretty=format:'%h %s' $range 2>/dev/null || true)"
  if [ -z "$log" ]; then
    echo "_（无提交）_"
    echo
    return
  fi
  local groups=("新功能|首发|新增|接入|支持|实现|feat"
                "修复|修复|修正|修 |bug|fix|回归"
                "性能|性能|优化|提速|perf"
                "重构与清理|重构|清理|剥离|卫生|死代码|refactor"
                "文档|文档|README|SYNTAX|注释|docs"
                "构建与工具链|CI|workflow|构建|发布|版本|脚本|依赖|测试")
  local titles=("新功能" "修复" "性能" "重构与清理" "文档" "构建与工具链")
  local assigned=()
  local i
  for i in "${!groups[@]}"; do
    local pattern="${groups[$i]}"
    local title="${titles[$i]}"
    local lines=""
    while IFS= read -r line; do
      [ -z "$line" ] && continue
      local subject="${line#* }"
      local sha="${line%% *}"
      local skip=0
      local j
      for j in $(seq 0 $((i - 1))); do
        if printf '%s' "$subject" | grep -qE "${groups[$j]}"; then skip=1; break; fi
      done
      [ "$skip" = "1" ] && continue
      if printf '%s' "$subject" | grep -qE "$pattern"; then
        lines+="- ${subject}（\`${sha}\`）"$'\n'
      fi
    done <<< "$log"
    if [ -n "$lines" ]; then
      echo "### ${title}"
      echo
      printf '%s' "$lines"
      echo
    fi
    assigned+=("$lines")
  done

  # 未匹配任何分组的提交
  local other=""
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    local subject="${line#* }"
    local sha="${line%% *}"
    local matched=0
    local p
    for p in "${groups[@]}"; do
      if printf '%s' "$subject" | grep -qE "$p"; then matched=1; break; fi
    done
    if [ "$matched" = "0" ]; then
      other+="- ${subject}（\`${sha}\`）"$'\n'
    fi
  done <<< "$log"
  if [ -n "$other" ]; then
    echo "### 其它"
    echo
    printf '%s' "$other"
    echo
  fi
}

if [ "${1:-}" = "--all" ]; then
  echo "# 变更日志"
  echo
  echo "> 由 \`scripts/changelog.sh --all\` 从 git 历史生成（分组：新功能 / 修复 / 性能 / 重构与清理 / 文档 / 构建与工具链 / 其它）。"
  echo
  prev=""
  for tag in $(git tag --sort=creatordate); do
    echo "## ${tag}"
    echo
    emit_commits "${prev:+${prev}..}${tag}"
    prev="$tag"
  done
  if [ -n "$prev" ]; then
    echo "## 未发布（${prev} 之后）"
    echo
    emit_commits "${prev}..HEAD"
  fi
  exit 0
fi

ARG="${1:-$(cat VERSION 2>/dev/null || echo unreleased)}"
START="${2:-$(git describe --tags --abbrev=0 2>/dev/null || echo '')}"
DATE="$(date -u '+%Y-%m-%d')"

echo "## ${ARG} - ${DATE}"
echo
emit_commits "${START:+${START}..}HEAD"
