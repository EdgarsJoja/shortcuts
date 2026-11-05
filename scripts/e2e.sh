#!/usr/bin/env bash
# End-to-end smoke test. Uses a throwaway HOME/XDG environment and never
# touches the caller's real configuration.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

export HOME="$TMP/home"
export XDG_CONFIG_HOME="$TMP/config"
export XDG_DATA_HOME="$TMP/data"
mkdir -p "$HOME" "$TMP/bin"

go build -o "$TMP/bin/shortcuts" "$ROOT"
BIN="$TMP/bin/shortcuts"
NEW_BIN="$TMP/bin/shortcuts-new"
SHIM_DIR="$XDG_DATA_HOME/shortcuts/bin"

die() { echo "FAIL: $*" >&2; exit 1; }

expect_code() {
  local want=$1
  shift
  set +e
  "$@" >/dev/null 2>&1
  local got=$?
  set -e
  [[ $got -eq $want ]] || die "expected exit $want, got $got: $*"
}

"$BIN" init --print >/dev/null

"$BIN" add greet -c 'printf "hello %s\n" {{1}}' -d "Greet someone"
"$BIN" add direct-ls --argv 'ls -lah' -d "Verbose listing"
"$BIN" add bad-exit -c 'exit 7'

[[ -L "$SHIM_DIR/greet" ]] || die "greet shim is missing or not a symlink"
[[ "$(readlink "$SHIM_DIR/greet")" == "$BIN" ]] || die "greet shim has wrong target"

export PATH="$SHIM_DIR:$PATH"

[[ "$(greet world)" == "hello world" ]] || die "greet output mismatch"
[[ "$(greet '; echo pwned')" == "hello ; echo pwned" ]] || die "argument injection"

"$BIN" run greet --dry-run world | grep -q "hello %s" || die "dry-run output"

"$BIN" list --json | grep -q '"name": "greet"' || die "list --json"
"$BIN" doctor --json >/dev/null
"$BIN" doctor | grep -q '^ok$' || die "doctor not clean"

ln -s "$BIN" "$TMP/bin/unknown-name"
expect_code 7 bad-exit
expect_code 127 "$TMP/bin/unknown-name"

"$BIN" edit greet -d "Updated description"
"$BIN" show greet | grep -q 'Updated description' || die "edit/show"

"$BIN" theme dracula
"$BIN" theme | grep -q '^\* dracula$' || die "theme listing"
grep -q 'theme = "dracula"' "$XDG_CONFIG_HOME/shortcuts/config.toml" || die "theme not persisted"
"$BIN" theme default >/dev/null

"$BIN" add killself -c 'kill -TERM $$'
expect_code 143 killself

# Binary relocation: move the executable and repair managed links.
mv "$BIN" "$NEW_BIN"
"$NEW_BIN" doctor --fix >/dev/null
[[ "$(readlink "$SHIM_DIR/greet")" == "$NEW_BIN" ]] || die "relocation repair failed"
[[ "$(greet again)" == "hello again" ]] || die "greet after relocation"

"$NEW_BIN" rm greet --force
[[ ! -e "$SHIM_DIR/greet" ]] || die "greet shim not removed"

echo "e2e: ok"
