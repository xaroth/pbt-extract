# pbt-extract

Reads Path of Exile game files and builds the datasets that [pbt-engine](https://github.com/xaroth/pbt-engine) loads.

Status: early. It reads files, tables and object template stats from a game install or `Bundles2` archive, and builds pbt datasets from them.

## Use

Requires Go 1.27 and [ooz](https://github.com/zao/ooz) (GPL-3.0, run as a separate program) on `PATH` or given with `-ooz`.

```sh
go run . -ooz /path/to/ooz get /path/to/game Data/Stats.datc64 > Stats.datc64
go run . -ooz /path/to/ooz table /path/to/game 3.29.3.3 Stats > Stats.jsonl
go run . -ooz /path/to/ooz stats /path/to/game Metadata/Characters/Str/Str
go run . -ooz /path/to/ooz dataset /path/to/game 3.29.3.3 /path/to/tree-export out/3.29.3.3
```

`table` decodes a table with the dat-schema build stored for that game version and writes one JSON object per row. `stats` merges the `Stats` block of an object template (`.ot`) over its `extends` chain and writes it as one JSON object. `dataset` writes a pbt dataset (format in [pbt-engine](https://github.com/xaroth/pbt-engine), `dataset` package) for that game version into the output directory. The tree export directory holds `data.json`, `ruthless.json` and `alternate.json` from [skilltree-export](https://github.com/grindinggear/skilltree-export) at the version's `treeExport` tag.

## Game versions

`versions/<game>/<version>/` holds `schema.json` (the dat-schema build that matches the version) and `source.json` (Steam manifest, dat-schema commit and skilltree-export tag). `versions/<game>/constants.json` holds edition constants the game data does not contain, each with a note and source. Add a version with:

```sh
scripts/add-version.sh poe1 <version> <steam-manifest> <dat-schema-commit> <tree-export-tag>
```

It needs git, node and npm. Pick the dat-schema commit closest after the patch, and check that the tables you read decode without a row size error.

## Develop

```sh
go build ./...
go test ./...
golangci-lint run
```

## Licence

Apache-2.0, see [LICENSE](LICENSE) and [NOTICE](NOTICE).

This product isn't affiliated with or endorsed by Grinding Gear Games in any way. Path of Exile and its game data are the property of Grinding Gear Games.
