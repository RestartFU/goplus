#!/usr/bin/env bash
# Copyright 2026 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
prefix=${GOPLUS_PREFIX:-"${HOME}/.local/share/goplus"}
bin_dir=${GOPLUS_BIN_DIR:-"${HOME}/.local/bin"}
tools_repo=${GOPLUS_TOOLS_REPO:-https://go.googlesource.com/tools}
tools_ref=${GOPLUS_TOOLS_REF:-635ae9663724}
gopls_ref=${GOPLUS_GOPLS_REF:-gopls/v0.22.0}

usage() {
	cat <<'EOF'
Usage: ./install-goplus.sh [--prefix DIR] [--bin-dir DIR]

Build and install the private Go+ toolchain plus:
  go+  gofmt+  gopls+  goimports+

Defaults:
  --prefix  $HOME/.local/share/goplus
  --bin-dir $HOME/.local/bin

Environment overrides: GOPLUS_PREFIX, GOPLUS_BIN_DIR, GOPLUS_TOOLS_REPO,
GOPLUS_TOOLS_REF, and GOPLUS_GOPLS_REF.
EOF
}

while (($#)); do
	case $1 in
	--prefix)
		(($# >= 2)) || { echo "--prefix requires a directory" >&2; exit 2; }
		prefix=$2
		shift 2
		;;
	--bin-dir)
		(($# >= 2)) || { echo "--bin-dir requires a directory" >&2; exit 2; }
		bin_dir=$2
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "unknown option: $1" >&2
		usage >&2
		exit 2
		;;
	esac
done

for command in bash git cc; do
	command -v "$command" >/dev/null 2>&1 || {
		echo "missing required command: $command" >&2
		exit 1
	}
done

mkdir -p "$(dirname -- "$prefix")" "$bin_dir"
prefix_parent=$(CDPATH= cd -- "$(dirname -- "$prefix")" && pwd -P)
prefix=$prefix_parent/$(basename -- "$prefix")
bin_dir=$(CDPATH= cd -- "$bin_dir" && pwd -P)
case $prefix in
"" | / | "$HOME")
	echo "refusing unsafe install prefix: $prefix" >&2
	exit 1
	;;
esac

echo "Building Go+ from $repo_root"
(cd "$repo_root/src" && ./make.bash)
test -x "$repo_root/bin/go"
test -x "$repo_root/bin/gofmt"
test -f "$repo_root/VERSION.cache"

work=$(mktemp -d "$prefix_parent/.goplus-install.XXXXXX")
trap 'rm -rf -- "$work"' EXIT
stage=$work/root
mkdir -p "$stage/go" "$stage/bin" "$stage/libexec"

for dir in api bin doc lib misc src; do
	cp -a "$repo_root/$dir" "$stage/go/"
done
mkdir -p "$stage/go/pkg"
cp -a "$repo_root/pkg/include" "$repo_root/pkg/tool" "$stage/go/pkg/"
cp -a "$repo_root/go.env" "$stage/go/go.env"
cp -a "$repo_root/VERSION.cache" "$stage/go/VERSION"

echo "Fetching patched x/tools ($tools_ref)"
git clone --quiet --filter=blob:none --no-checkout "$tools_repo" "$work/tools"
git -C "$work/tools" checkout --quiet "$tools_ref"
git -C "$work/tools" apply "$repo_root/misc/enum/x-tools.patch"

echo "Fetching patched gopls ($gopls_ref)"
git clone --quiet --filter=blob:none --no-checkout "$tools_repo" "$work/gopls-repo"
git -C "$work/gopls-repo" checkout --quiet "$gopls_ref"
gopls_dir=$work/gopls-repo/gopls
git -C "$work/gopls-repo" apply --directory=gopls "$repo_root/misc/enum/gopls.patch"

private_go=$stage/go/bin/go
export GOROOT=$stage/go
export GOTOOLCHAIN=local
export GOWORK=off

(cd "$work/tools" && "$private_go" build -trimpath -o "$stage/libexec/goimports" ./cmd/goimports)
(cd "$gopls_dir" && "$private_go" mod edit -replace="golang.org/x/tools=$work/tools")
(cd "$gopls_dir" && "$private_go" build -trimpath -o "$stage/libexec/gopls" .)

write_launcher() {
	local name=$1 target=$2
	cat >"$stage/bin/$name" <<EOF
#!/bin/sh
set -eu
self=\$(readlink -f -- "\$0")
root=\$(CDPATH= cd -- "\$(dirname -- "\$self")/.." && pwd -P)
export GOROOT="\$root/go"
export GOTOOLCHAIN=local
export PATH="\$GOROOT/bin:\$PATH"
exec "\$root/$target" "\$@"
EOF
	chmod 0755 "$stage/bin/$name"
}

write_launcher go+ go/bin/go
write_launcher gofmt+ go/bin/gofmt
write_launcher gopls+ libexec/gopls
write_launcher goimports+ libexec/goimports

"$stage/bin/go+" version
"$stage/bin/gopls+" version

backup=
if [[ -e $prefix ]]; then
	backup=$prefix.backup.$$
	test ! -e "$backup"
	mv -- "$prefix" "$backup"
fi
if ! mv -- "$stage" "$prefix"; then
	[[ -z $backup ]] || mv -- "$backup" "$prefix"
	exit 1
fi
[[ -z $backup ]] || rm -rf -- "$backup"

if [[ $bin_dir != "$prefix/bin" ]]; then
	for name in go+ gofmt+ gopls+ goimports+; do
		ln -sfn "$prefix/bin/$name" "$bin_dir/$name"
	done
fi

echo
echo "Go+ installed in $prefix"
echo "Commands linked into $bin_dir: go+, gofmt+, gopls+, goimports+"
case :$PATH: in
*:"$bin_dir":*) ;;
*) echo "Add $bin_dir to PATH before using the commands." ;;
esac
