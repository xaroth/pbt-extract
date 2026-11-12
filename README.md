# pbt-extract

Reads Path of Exile game files and builds the datasets that [pbt-engine](https://github.com/xaroth/pbt-engine) loads.

Status: early. It reads files, tables and object template stats from a game install or `Bundles2` archive, and builds pbt datasets from them.

## Use

Requires Go 1.27 and [ooz](https://github.com/zao/ooz) (GPL-3.0, run as a separate program) on `PATH` or given with `-ooz`.

```sh
go run . -version 3.29.3.3 dataset <in> <out>   # build the dataset into out
go run . art <in> <out>             # the tree's sprite sheets and their manifest into out
go run . bundles <in>               # bundle files a build may read, from Bundles2/_.index.bin alone
go run . get <in> Data/Stats.datc64 > Stats.datc64
go run . table <in> Stats > Stats.jsonl
go run . stats <in> Metadata/Characters/Str/Str
```

`in` is an input directory; pbt-extract does not fetch anything. `-version` names the game version the files are from and `-game` the edition (default `poe1`).

| Entry | Content |
|---|---|
| `schema.json` | The [dat-schema](https://github.com/poe-tool-dev/dat-schema) build for the version |
| `tree/` | `data.json`, `ruthless.json` and `alternate.json` from [skilltree-export](https://github.com/grindinggear/skilltree-export), and its `assets/` folder for `art` |
| `Bundles2/` | `_.index.bin` and the bundles `bundles` lists |

`art` copies the sheets the standard tree's sprites name and writes `manifest.json`: each sheet's name, size and SHA-256, and per sprite kind (such as `normalActive`) one sheet per zoom level with each icon's rectangle. It fails when an icon or group background of any variant is not on the sheets of its kind at every zoom level: a node's icon is on the `keystone`, `notable`, `mastery` or `normal` sheets by node type, active and inactive. The class illustrations the layouts name are not in the export's sheets.

Which game version uses which depot manifest, dat-schema commit and tree export commit is kept in [pbt-data](https://github.com/xaroth/pbt-data), which builds and publishes the datasets.

`edition/<game>/` holds what the game data of an edition does not say, each entry with a note and source: `constants.json` for constants, `columns.json` for our names of columns dat-schema leaves unnamed, `totems.json` for the totems of supports, a link the game data does not hold; `values.json` for how the game files of older leagues number a column whose values the dat-schema enumeration numbers otherwise.

## Develop

```sh
go build ./...
go test ./...
golangci-lint run
```

## Licence

Apache-2.0, see [LICENSE](LICENSE) and [NOTICE](NOTICE).

This product isn't affiliated with or endorsed by Grinding Gear Games in any way. Path of Exile and its game data are the property of Grinding Gear Games.
