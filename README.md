# pbt-extract

Reads Path of Exile game files and builds the datasets that [pbt-engine](https://github.com/xaroth/pbt-engine) loads.

Status: early. It reads files, tables and object template stats from a game install or `Bundles2` archive, and builds pbt datasets from them.

## Use

Requires Go 1.27 and [ooz](https://github.com/zao/ooz) (GPL-3.0, run as a separate program) on `PATH` or given with `-ooz`.

```sh
go run . dataset <in> <out>         # build the dataset into out
go run . bundles <in>               # bundle files a build may read, from Bundles2/_.index.bin alone
go run . get <in> Data/Stats.datc64 > Stats.datc64
go run . table <in> Stats > Stats.jsonl
go run . stats <in> Metadata/Characters/Str/Str
```

`in` is an input directory; pbt-extract does not fetch anything:

| Entry | Content |
|---|---|
| `source.json` | `{"version": "3.29.3.3", ...}`: string fields, copied into the dataset manifest's `source` as given; `-game` names the edition (default `poe1`) |
| `schema.json` | The [dat-schema](https://github.com/poe-tool-dev/dat-schema) build for the version |
| `tree/` | `data.json`, `ruthless.json` and `alternate.json` from [skilltree-export](https://github.com/grindinggear/skilltree-export) |
| `Bundles2/` | `_.index.bin` and the bundles `bundles` lists |

Which game version uses which depot manifest, dat-schema commit and tree export commit is kept in [pbt-data](https://github.com/xaroth/pbt-data), which builds and publishes the datasets.

`constants/<game>.json` holds edition constants the game data does not contain, each with a note and source.

## Develop

```sh
go build ./...
go test ./...
golangci-lint run
```

## Licence

Apache-2.0, see [LICENSE](LICENSE) and [NOTICE](NOTICE).

This product isn't affiliated with or endorsed by Grinding Gear Games in any way. Path of Exile and its game data are the property of Grinding Gear Games.
