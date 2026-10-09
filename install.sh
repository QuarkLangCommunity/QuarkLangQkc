#!/bin/sh
# QuarkLang toolchain installer.
#
# Installs the whole toolchain — the interpreter (quark), the native compiler (qkc), the
# command-line tools (qkcheck, qkdoc, qkrepl, qklsp) and the sibling tools (qkfmt, qktest,
# qkm, qkd the debugger) — plus the standard library, the specification and the examples.
#
# Two ways to get the core binaries:
#   prebuilt  download quark/qkc from the latest GitHub release for this platform
#   source    build everything from source (needs Go >= 1.26; the extra tools are always built
#             from source, because only quark and qkc are published as release assets)
#
# Usage:
#   ./install.sh                       # auto: source mode inside a checkout, prebuilt otherwise
#   ./install.sh --from-source         # build everything from source
#   ./install.sh --prebuilt            # download the core, build the extra tools from source
#   ./install.sh --prefix ~/.local     # where to install (default: /usr/local as root, else ~/.local)
#   ./install.sh --version v2.1.0      # pin a release tag (default: latest)
#   ./install.sh --only core           # core | tools | all
#   ./install.sh --dry-run             # print every action, change nothing
#
# The script never writes to /tmp: build and clone work happens under
# ${XDG_CACHE_HOME:-$HOME/.cache}/quarklang.
set -eu

ORG="QuarkLangCommunity"
CORE_REPO="QuarkLangQkc"
MIN_GO_MAJOR=1
MIN_GO_MINOR=26

# Sibling repositories that hold the extra tools: "<repo>:<binary>".
SIBLING_TOOLS="QuarkLangQkfmt:qkfmt QuarkLangQktest:qktest QuarkLangQkm:qkm QuarkLangQkd:qkd"

MODE="auto"
ONLY="all"
VERSION="latest"
PREFIX=""
DRY_RUN=0

say() { printf '%s\n' "$*"; }
die() { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

# run executes a command, or prints it in --dry-run mode.
run() {
	if [ "$DRY_RUN" = 1 ]; then
		printf '  [dry-run] %s\n' "$*"
		return 0
	fi
	"$@"
}

usage() {
	sed -n '2,26p' "$0" | sed 's/^# \{0,1\}//'
}

parse_args() {
	while [ $# -gt 0 ]; do
		case "$1" in
		--prefix) PREFIX="${2:?--prefix needs a directory}"; shift 2 ;;
		--prefix=*) PREFIX="${1#*=}"; shift ;;
		--version) VERSION="${2:?--version needs a tag}"; shift 2 ;;
		--version=*) VERSION="${1#*=}"; shift ;;
		--only) ONLY="${2:?--only needs core|tools|all}"; shift 2 ;;
		--only=*) ONLY="${1#*=}"; shift ;;
		--from-source) MODE="source"; shift ;;
		--prebuilt) MODE="prebuilt"; shift ;;
		--dry-run) DRY_RUN=1; shift ;;
		-h | --help) usage; exit 0 ;;
		*) die "unknown option: $1 (try --help)" ;;
		esac
	done
	case "$ONLY" in core | tools | all) ;; *) die "--only must be core, tools or all" ;; esac
}

# repo_root is the checkout this script lives in, or empty when run from a copy alone.
repo_root() {
	dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
	if [ -f "$dir/go.mod" ] && [ -f "$dir/main.go" ]; then
		printf '%s' "$dir"
	fi
}

# detect_platform maps uname onto the GOOS/GOARCH pair used by the release assets.
detect_platform() {
	os=$(uname -s)
	case "$os" in
	Linux) GOOS="linux" ;;
	Darwin) GOOS="darwin" ;;
	MINGW* | MSYS* | CYGWIN*) GOOS="windows" ;;
	*) die "unsupported operating system: $os" ;;
	esac
	arch=$(uname -m)
	case "$arch" in
	x86_64 | amd64) GOARCH="amd64" ;;
	aarch64 | arm64) GOARCH="arm64" ;;
	*) die "unsupported architecture: $arch" ;;
	esac
	EXT=""
	if [ "$GOOS" = "windows" ]; then EXT=".exe"; fi
}

# default_prefix installs into /usr/local for root and into ~/.local otherwise.
default_prefix() {
	if [ -n "$PREFIX" ]; then return 0; fi
	if [ "$(id -u)" = "0" ]; then PREFIX="/usr/local"; else PREFIX="$HOME/.local"; fi
}

# cache_dir keeps every download and build outside /tmp.
cache_dir() {
	base="${XDG_CACHE_HOME:-$HOME/.cache}"
	printf '%s/quarklang' "$base"
}

check_go() {
	command -v go >/dev/null 2>&1 || die "Go is required to build from source (https://go.dev/dl)"
	have=$(go env GOVERSION 2>/dev/null | sed 's/^go//')
	major=${have%%.*}
	rest=${have#*.}
	minor=${rest%%.*}
	if [ "${major:-0}" -lt "$MIN_GO_MAJOR" ] || { [ "${major:-0}" -eq "$MIN_GO_MAJOR" ] && [ "${minor:-0}" -lt "$MIN_GO_MINOR" ]; }; then
		die "Go >= ${MIN_GO_MAJOR}.${MIN_GO_MINOR} is required, found ${have:-unknown}"
	fi
}

# release_url builds the download URL for one asset of the requested release.
release_url() {
	asset="$1"
	base="https://github.com/${ORG}/${CORE_REPO}/releases"
	if [ "$VERSION" = "latest" ]; then
		printf '%s/latest/download/%s' "$base" "$asset"
	else
		printf '%s/download/%s/%s' "$base" "$VERSION" "$asset"
	fi
}

# checksum prints the sha256 of a file with whichever tool exists.
checksum() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d' ' -f1
	else
		printf 'unavailable'
	fi
}

# download_core fetches quark and qkc from the release for this platform.
download_core() {
	for name in quark qkc; do
		asset="${name}-${GOOS}-${GOARCH}${EXT}"
		out="${BIN}/${name}${EXT}"
		say "→ downloading ${asset}"
		run curl -fsSL "$(release_url "$asset")" -o "$out"
		run chmod +x "$out"
		[ "$DRY_RUN" = 1 ] || say "  sha256 $(checksum "$out")"
	done
}

# build_core compiles quark, the cmd tools and qkc from a checkout.
build_core() {
	root="$1"
	say "→ building quark"
	run sh -c "cd '$root' && go build -trimpath -ldflags '-s -w' -o '${BIN}/quark${EXT}' ."
	for tool in qkcheck qkdoc qkrepl qklsp; do
		say "→ building ${tool}"
		run sh -c "cd '$root' && go build -trimpath -ldflags '-s -w' -o '${BIN}/${tool}${EXT}' ./cmd/${tool}"
	done
	say "→ building qkc"
	run sh -c "cd '$root/compiler' && go build -trimpath -ldflags '-s -w' -o '${BIN}/qkc${EXT}' ."
}

# link_core_into_workspace exposes the core checkout as <workspace>/QuarkLang, the path the
# sibling go.mod files replace (`replace github.com/QuarkLangCommunity/QuarkLangQkc => ../QuarkLang`) expect to find.
link_core_into_workspace() {
	ws="$1"
	root="$2"
	if [ "$DRY_RUN" = 1 ]; then return 0; fi
	mkdir -p "$ws"
	if [ -n "$root" ]; then
		ln -sfn "$root" "${ws}/QuarkLang"
	elif [ ! -d "${ws}/QuarkLang" ]; then
		say "→ cloning ${CORE_REPO} (the sibling tools resolve their parser through it)"
		git clone --depth 1 "https://github.com/${ORG}/${CORE_REPO}.git" "${ws}/QuarkLang"
	fi
}

# ensure_sibling returns the workspace path of a sibling repository, linking the local checkout when
# there is one and cloning it otherwise.
ensure_sibling() {
	repo="$1"
	root="$2"
	dir="$(cache_dir)/src/${repo}"
	if [ ! -e "$dir" ]; then
		if [ -n "$root" ] && [ -d "${root}/${repo}" ]; then
			run ln -sfn "${root}/${repo}" "$dir"
		else
			say "→ cloning ${repo}"
			run git clone --depth 1 "https://github.com/${ORG}/${repo}.git" "$dir"
		fi
	fi
	printf '%s' "$dir"
}

# build_siblings compiles the tools that live in their own repositories.
build_siblings() {
	root="$1"
	link_core_into_workspace "$(cache_dir)/src" "$root"
	for entry in $SIBLING_TOOLS; do
		repo=${entry%%:*}
		tool=${entry##*:}
		dir=$(ensure_sibling "$repo" "$root")
		say "→ building ${tool} (${repo})"
		run sh -c "cd '$dir' && go build -trimpath -ldflags '-s -w' -o '${BIN}/${tool}${EXT}' ."
	done
}

# install_share copies the standard library, the spec and the examples under the prefix.
install_share() {
	root="$1"
	[ -n "$root" ] || return 0
	for pair in "stdlib:stdlib" "spec:spec" "examples:examples"; do
		src="${root}/${pair%%:*}"
		dst="${SHARE}/${pair##*:}"
		[ -d "$src" ] || continue
		say "→ installing $(basename "$src")"
		run mkdir -p "$dst"
		run cp -R "${src}/." "$dst/"
	done
}

# write_env_hint tells the user how to reach the toolchain from a shell.
write_env_hint() {
	if [ "$DRY_RUN" = 1 ]; then return 0; fi
	cat >"${SHARE}/env.sh" <<EOF
# Add the QuarkLang toolchain to this shell:
export PATH="${BIN}:\$PATH"
export QK_STDLIB="${SHARE}/stdlib"
EOF
}

# verify_install runs the installed interpreter once, when not dry-running.
verify_install() {
	if [ "$DRY_RUN" = 1 ]; then return 0; fi
	if [ ! -x "${BIN}/quark${EXT}" ]; then return 0; fi
	say "→ ${BIN}/quark --version"
	"${BIN}/quark${EXT}" --version 2>/dev/null || say "  (the interpreter does not report a version)"
}

main() {
	parse_args "$@"
	detect_platform
	default_prefix
	BIN="${PREFIX}/bin"
	SHARE="${PREFIX}/share/quarklang"
	root=$(repo_root)

	if [ "$MODE" = "auto" ]; then
		if [ -n "$root" ]; then MODE="source"; else MODE="prebuilt"; fi
	fi
	say "QuarkLang installer: prefix=${PREFIX} platform=${GOOS}/${GOARCH} mode=${MODE} only=${ONLY}"

	run mkdir -p "$BIN" "$SHARE"

	if [ "$ONLY" != "tools" ]; then
		if [ "$MODE" = "source" ]; then
			[ -n "$root" ] || die "--from-source must run from a checkout of ${CORE_REPO}"
			check_go
			build_core "$root"
		else
			download_core
		fi
	fi

	if [ "$ONLY" != "core" ]; then
		check_go
		build_siblings "$root"
	fi

	install_share "$root"
	write_env_hint
	verify_install

	say ""
	say "✓ installed into ${PREFIX}"
	say "  binaries: ${BIN}"
	say "  data:     ${SHARE}"
	say ""
	say "Add it to your shell:"
	say "  . ${SHARE}/env.sh"
}

main "$@"
