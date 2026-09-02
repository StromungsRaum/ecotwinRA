# ECOTWIN Controller `twinctl`

`twinctl` is the command-line interface for inspecting and working with ECOTWIN twin
configurations. It reads a YAML file (default `twin.yaml`) and exposes several
output formats for exploring the asset structure.

## Getting Started

`twinctl` is distributed as a PEP 723 inline script (`ecotwin/scripts/twinctl.py`). The
simplest way to run it is through [uv](https://docs.astral.sh/uv/), which creates an
ephemeral virtual environment on demand — this is the only supported invocation, there is
no installed `twinctl`/`bin/twinctl` console script:

```shell
cd ecotwinRA
uv run ecotwin/scripts/twinctl.py --help
```

Top-level subcommand groups: `info` (inspect twin config / query StrömungsRaum models),
`login` (store SR credentials), and `submission` (submit a full model for simulation).

## `info system` Subcommand

`twinctl info system` renders the parsed twin configuration using several presentation styles.
The `--file`/`--format` options belong to this subcommand, not to bare `info`:

```shell
uv run ecotwin/scripts/twinctl.py info system [--file path/to/twin.yaml] [--format raw|tree|markdown-tree|markdown-files]
```

- `raw` (default) pretty-prints the configuration using `beeprint`.
- `tree` outputs a simple indented hierarchy, suitable for terminal inspection.
- `markdown-tree` mirrors the hierarchy as Markdown bullet lists.
- `markdown-files` builds a set of Markdown documents that link industrial
  processes → digital twins → applications → models.

### Tree Views

Use the hierarchy options to quickly understand the nesting inside the YAML file.

```shell
uv run ecotwin/scripts/twinctl.py info system --format tree
```

```shell
uv run ecotwin/scripts/twinctl.py info system --format markdown-tree
```

Both commands include every key in the YAML document. The Markdown variant wraps
keys in bold for legibility and can be copy-pasted into documentation.

### Generating Markdown Files

`markdown-files` creates individual Markdown documents beneath a chosen root.
It is ideal when you need lightweight docs for each process, digital twin,
application, and model.

```shell
uv run ecotwin/scripts/twinctl.py info system \
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

## `info models` Subcommand

`twinctl info models` queries the StrömungsRaum backend (`ianus-simod`) for its list of
submission campaigns (`GET /api/automated_user/submission/campaigns`), independently of
Platform Mesh/kcp — this is the standalone discovery client for that API. Each result's
`var` is the `--campaign-var` value `submission submit` (below) expects.

```shell
uv run ecotwin/scripts/twinctl.py info models --system int
```

`--system` selects which backend to hit (`prod`/`int`/`test`/`dev`); credentials come from
`EMAIL`/`PASSWD` (or `EMAIL_<SYSTEM>`/`PASSWD_<SYSTEM>`) env vars — see `login store` below.

The legacy, session/CSRF-based `/form/ajax` scraping path (superseded by the endpoint
above) still works via `create_api_handler(system, legacy=True)` in
`ecotwin/common/api_connector.py`, but is not exposed as its own `twinctl` flag.

## `submission submit` Subcommand

`twinctl submission submit` submits geometry + process/material/reporter parameters in
one atomic call (`POST /api/automated_user/submission`): builds the digital twin and
creates the simulation server-side — the standalone smoke-test client for that API,
independently of Platform Mesh/kcp. This replaces the old `/form/ajax`-scraping
integration as the intended way to submit a model, for campaigns discovered via
`info models` above.

```shell
uv run ecotwin/scripts/twinctl.py submission submit \
  --campaign-var injection-molding-v1 \
  --params-file params.json \
  --system int
```

`--params-file` is a JSON file with the request body (minus `campaign_var`, which
`--campaign-var` supplies): `geometry` (`component_description_uuid` + `parameters`),
`process_parameters`, `material_selections`, `reporter_parameters`, and optionally `name`
and `project_id`. See `docs/api/aideas.md` in the `ianus-simod` repo for the full request/
response shape.

## `login store` Subcommand

`twinctl login store` persists StrömungsRaum credentials to `~/.config/ecotwin/.env` so
`info models` (and other commands hitting the backend) don't need env vars set every time:

```shell
uv run ecotwin/scripts/twinctl.py login store --system int --email you@example.com --pass '...'
```

## Capability Reference

Detailed playbooks for provisioning and operating twin workloads live alongside the
controller:

- [Provisioning (Control Plane)](capabilities/provisioning.md)
- [Operations (Runtime)](capabilities/operations.md)
