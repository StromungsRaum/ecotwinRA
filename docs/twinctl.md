# ECOTWIN Controller `twinctl`

`twinctl` is the command-line interface for inspecting and working with ECOTWIN twin
configurations. It reads a YAML file (default `twin.yaml`) and exposes several
output formats for exploring the asset structure.

## Getting Started

`twinctl` is distributed as an inline script. The simplest way to run it is through
[uv](https://docs.astral.sh/uv/), which will create an ephemeral virtual
environment on demand.

```shell
bin/twinctl
```

You can also manage the environment yourself:

```shell
cd ecotwinRA
uv sync
source .venv/bin/activate
twinctl
```

The CLI accepts a `--file` option (see `twinctl --help`) to load an alternative
YAML document.

## `info` Subcommand

`twinctl info` renders the parsed configuration using several presentation styles.

```shell
twinctl info [--file path/to/twin.yaml] [--format raw|tree|markdown-tree|markdown-files]
```

- `raw` (default) pretty-prints the configuration using `beeprint`.
- `tree` outputs a simple indented hierarchy, suitable for terminal inspection.
- `markdown-tree` mirrors the hierarchy as Markdown bullet lists.
- `markdown-files` builds a set of Markdown documents that link industrial
  processes → digital twins → applications → models.

### Tree Views

Use the hierarchy options to quickly understand the nesting inside the YAML file.

```shell
twinctl info --format tree
```

```shell
twinctl info --format markdown-tree
```

Both commands include every key in the YAML document. The Markdown variant wraps
keys in bold for legibility and can be copy-pasted into documentation.

### Generating Markdown Files

`markdown-files` creates individual Markdown documents beneath a chosen root.
It is ideal when you need lightweight docs for each process, digital twin,
application, and model.

```shell
twinctl info \
  --format markdown-files \
  --root-name "Pump Suite" \
  --output-dir docs/generated
```

Options:

- `--root-name` (required) becomes the title of the root document.
- `--output-dir` sets the destination directory (defaults to the current working directory).

`twinctl` creates the root Markdown file named after the provided root title.
Each industrial process referenced in `industrial_processes` receives its own
document. Entity pages include a `## Linked Entities` section containing every
referenced ID discovered anywhere in the entity’s fields (lists, mappings, nested
structures). That means you automatically get links from processes to digital twins,
from twins to applications, sensors, and beyond—whatever the YAML graph contains.
Filenames derive from the `name` field of each entity (falling back to the ID if no
name exists), normalized for filesystem safety. The generator warns in the terminal
if a referenced entity is missing from the configuration.

After generation you can use the resulting Markdown tree directly in user-facing
documentation or feed it into static site tooling. The process is idempotent—run
it whenever the YAML changes to keep the documentation in sync.

## Capability Reference

Detailed playbooks for provisioning and operating twin workloads live alongside the
controller:

- [Provisioning (Control Plane)](twinctl/capabilities/provisioning.md)
- [Operations (Runtime)](twinctl/capabilities/operations.md)
