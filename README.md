# llrm

llrm is a CLI tool to clean up AWS Lambda Layer versions.

Each layer publish creates a new version, and old ones must be deleted individually — llrm finds all versions not attached to any function and lets you delete them in bulk.

## Installation

### mise

```bash
mise use -g go:github.com/miztch/llrm@latest
```

### go install

```bash
go install github.com/miztch/llrm@latest
```

## Usage

```bash
# List unused layer versions (candidates for deletion)
llrm --list

# List all layer versions with attachment status
llrm --list-all

# Delete unused layer versions (with confirmation prompt)
llrm

# Skip confirmation prompt
llrm --yes
```

## Flags

| Flag              | Default | Description                                              |
|-------------------|---------|----------------------------------------------------------|
| `--region`        | (env)   | AWS region (falls back to `AWS_DEFAULT_REGION` / config) |
| `--name`          | (none)  | Target a specific layer by exact name                    |
| `--filter`        | (none)  | Filter layers by name (substring match)                  |
| `--keep-versions` | `0`     | Keep the N most recent versions per layer; delete older ones |
| `--list`          | `false` | Print candidates without deleting                        |
| `--list-all`      | `false` | Print all layer versions including attached ones         |
| `--yes`           | `false` | Skip confirmation prompt                                 |
| `--output`        | `table` | Output format: `table`, `json`, or `yaml`                |
