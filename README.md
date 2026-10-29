# sarifari

A terminal UI for browsing
[SARIF](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html) files
from static analysis tools:
finding counts by group, drill-down into a group,
and a detailed view with source context and rule help.

## Install

With [mise](https://mise.jdx.dev):

```sh
mise use -g go:github.com/peterchristofferholm/sarifari/cmd/sarifari@latest
```

Or with plain Go:

```sh
go install github.com/peterchristofferholm/sarifari/cmd/sarifari@latest
```

## Usage

```sh
opengrep scan --sarif --sarif-output=results.sarif .
sarifari --root . results.sarif
```

## SARIF support

sarifari only reads the parts of SARIF 2.1.0 that most tools emit,
not the full spec.
It is developed against [opengrep](https://github.com/opengrep/opengrep) output
and should work with any tool producing similar files.

Supported:
multiple runs,
rules matched by `ruleIndex` or `ruleId`,
result `level` with the rule's default as fallback,
the first physical location with its region and snippet,
and rule descriptions, help text or markdown, and `helpUri`.

Not supported:
additional locations,
`relatedLocations`,
`codeFlows`,
`fixes`,
`suppressions`,
`messageStrings` templates,
`uriBaseId` resolution,
and `run.artifacts`.
A version other than 2.1.0 still loads, with a warning.

## Flags

| Flag         | Default                              | Description                                                                   |
|--------------|--------------------------------------|-------------------------------------------------------------------------------|
| `--root`     | `.`, then the SARIF file's directory | Where source paths are resolved; absolute paths and `file://` URIs work as-is |
| `--group`    | `rule`                               | `rule`, `severity`, `file` or `dir`                                           |
| `--severity` | `all`                                | `all`, `error`, `warning` or `note`                                           |
| `--style`    | auto                                 | Chroma style; `github` or `github-dark` by terminal background                |

Exit codes:
0 normal,
1 load or parse error,
2 usage error.

## Keys

| Key                 | Action                                          |
|---------------------|-------------------------------------------------|
| `j` / `k`           | Move                                            |
| `enter`             | Open                                            |
| `esc` / `-`         | Back; quits on the summary                      |
| `tab` / `shift+tab` | Regroup (summary only)                          |
| `s`                 | Cycle the severity filter (summary only)        |
| `/`                 | Filter                                          |
| `e`                 | Open in editor                                  |
| `n` / `p`           | Next or previous finding (detail only)          |
| `?`                 | Help                                            |
| `q`                 | Quit                                            |
