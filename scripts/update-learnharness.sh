#!/usr/bin/env bash
# Bumps home/apps/learnharness.nix to the latest GitHub release. The hash comes
# from the sha256 digest GitHub publishes per asset, so nothing is downloaded.
# Under GitHub Actions the new version is written to $GITHUB_OUTPUT.
set -euo pipefail

repo=amosblomqvist/learnharness
file="$(dirname "$0")/../home/apps/learnharness.nix"

auth=()
[ -n "${GITHUB_TOKEN:-}" ] && auth=(-H "Authorization: Bearer $GITHUB_TOKEN")
release=$(curl -fsSL "${auth[@]}" "https://api.github.com/repos/$repo/releases/latest")

version=$(jq -r '.tag_name | ltrimstr("v")' <<<"$release")
current=$(sed -nE 's/^  version = "([^"]+)";/\1/p' "$file")

if [ "$version" = "$current" ]; then
  echo "learnharness is up to date ($current)"
  exit 0
fi

asset="LearnHarness-$version-x86_64.AppImage"
digest=$(jq -r --arg name "$asset" '.assets[] | select(.name == $name) | .digest // empty' <<<"$release")
if [ -z "$digest" ]; then
  echo "release v$version has no $asset digest" >&2
  exit 1
fi
hash=$(nix hash convert --hash-algo sha256 --to sri "${digest#sha256:}")

sed -i -E \
  -e "s|^(  version = \")[^\"]+|\1$version|" \
  -e "s|^(    hash = \")[^\"]+|\1$hash|" \
  "$file"

echo "learnharness $current -> $version"
echo "version=$version" >>"${GITHUB_OUTPUT:-/dev/null}"
