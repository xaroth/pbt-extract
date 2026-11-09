#!/usr/bin/env bash
# Adds versions/<game>/<version>/ with the dat-schema build for that version and its source.
# usage: scripts/add-version.sh <game> <version> <steam-manifest> <dat-schema-commit> <tree-export-tag>
# Needs git, node, npm and curl.
set -euo pipefail

if [[ $# -ne 5 ]]; then
	echo "usage: $0 <game> <version> <steam-manifest> <dat-schema-commit> <tree-export-tag>" >&2
	exit 2
fi
game=$1 version=$2 manifest=$3 commit=$4 tree=$5
case $game in
poe1) valid_for=1 ;;
poe2) valid_for=2 ;;
*) echo "unknown game $game" >&2; exit 2 ;;
esac

out=$(cd "$(dirname "$0")/.." && pwd)/versions/$game/$version
if [[ -e $out ]]; then
	echo "$out exists" >&2
	exit 1
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
git clone -q --filter=blob:none https://github.com/poe-tool-dev/dat-schema.git "$work/dat-schema"
cd "$work/dat-schema"
git -c advice.detachedHead=false checkout -q "$commit"
full=$(git rev-parse HEAD)
npm ci --ignore-scripts --no-audit --no-fund --loglevel=error
npm exec --no -- tsc
node dist/cli.js >/dev/null

mkdir -p "$out"
# Older dat-schema commits write only the combined schema.min.json, so filter by game here.
node -e '
const fs = require("fs");
const [src, dst, bit] = process.argv.slice(1);
const s = JSON.parse(fs.readFileSync(src, "utf8"));
const keep = (e) => (e.validFor & Number(bit)) !== 0;
fs.writeFileSync(dst, JSON.stringify({ ...s, tables: s.tables.filter(keep), enumerations: s.enumerations.filter(keep) }));
' schema.min.json "$out/schema.json" "$valid_for"
export_repo=https://github.com/grindinggear/skilltree-export
tree_commit=$(git ls-remote "$export_repo" "refs/tags/$tree^{}" "refs/tags/$tree" | head -1 | cut -f1)
if [[ -z $tree_commit ]]; then
	echo "no tag $tree in $export_repo" >&2
	exit 1
fi
sums=()
for f in alternate.json data.json ruthless.json; do
	sum=$(curl -fsSL "https://raw.githubusercontent.com/grindinggear/skilltree-export/$tree_commit/$f" | sha256sum | cut -c1-64)
	sums+=("    \"$f\": \"$sum\"")
done
files=$(IFS=,; printf '%s' "${sums[*]}" | sed 's/,/,\n/g')
printf '{\n  "steamManifest": "%s",\n  "datSchemaCommit": "%s",\n  "treeExport": "%s",\n  "treeExportCommit": "%s",\n  "treeExportFiles": {\n%b\n  }\n}\n' \
	"$manifest" "$full" "$tree" "$tree_commit" "$files" > "$out/source.json"
echo "wrote $out"
