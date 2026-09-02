# ECOTWIN Reference Architecture Agent Instructions

Use this file as the high-signal project guide for coding agents working in this repository. It
supplements [CLAUDE.md](CLAUDE.md) and [README.md](README.md); do not duplicate what's already stated
there — link to it instead.

Agents may update this file when they discover missing, outdated, or misleading project knowledge. Keep
additions factual, concise, and linked to the relevant source files so future agents can verify the context
quickly.

**Local repo convention:** the sibling backend this repo talks to is checked out at `~/repo/ianus-simod`
(this repo is `repo/ecotwinRA`). References below use the `repo/<name>` form so they read correctly
regardless of which file/depth they're quoted from — a real relative path from this repo's root would be
`../ianus-simod`.

## Where This Repo Sits In The Larger Pipeline

`ecotwin/common/api_connector.py` is a client of [repo/ianus-simod](../ianus-simod)'s Laravel backend
(`backend.{prod,int,test,dev}.simod.de`), not a standalone system:

- `POST /api/login` (token auth, `JobHandler`) — used by every non-legacy call.
- `GET /api/automated_user/submission/campaigns` — list the submission campaigns (product/application
  templates) available to the account's company (`ApiHandler.get_models()`, exposed as `twinctl info
  models`).
- `POST /api/automated_user/submission` — the "full submission" API: builds geometry, freezes a digital
  twin, creates a simulation, all in one call (`ApiHandler.submit()`, exposed as `twinctl submission
  submit`). This is the intended replacement for the legacy `/form/ajax` scraping integration.
- The legacy path (`BackendConnector` + `AjaxApiHandler`, session/CSRF login against the staff-only
  `admin/configuration/form/ajax` DataTables endpoint) is **kept working**, not deleted — pass
  `legacy=True` to `create_api_handler()` to use it. Don't remove it without an explicit ask; it was
  intentionally preserved during the migration to the token-based API even though it's no longer the
  default.

**`twinctl` is the standalone smoke-test client for `ianus-simod`'s automated-user API**, independent of
Platform Mesh/kcp entirely — `twinctl info models` and `twinctl submission submit` exercise the real
backend over plain HTTP with no cluster, no CRDs, no operator involved. This is deliberate: it lets you
verify the `ianus-simod` side of the integration works before touching anything kcp-related. See
[docs/twinctl/twinctl.md](docs/twinctl/twinctl.md) for the full CLI reference.

**Forward-looking, not yet wired up:** the `Model` CRD's `Spec.SecretRefs`
(`apis/ianus/v1alpha1/model_types.go`) is designed to eventually hold the credentials this same API needs,
so the operator (`operator/ianus/controller.go`) can drive submissions from a `Model` resource instead of
a human running `twinctl` by hand. The current reconciler is a placeholder (see [CLAUDE.md](CLAUDE.md)) and
does not call this API yet — `twinctl` is the only thing exercising it today.

**Manual sync point, no shared schema:** there is no contract test or codegen between the two repos. If
`ianus-simod`'s `SimulationSubmissionController` route or response shape changes, `ecotwin/common/api_connector.py`'s
`JobHandler`/`ApiHandler` need a matching update, and vice versa. Check
[repo/ianus-simod/agent_instructions.md](../ianus-simod/agent_instructions.md)'s "Full Submission API"
section (under Backend Architecture Notes) when touching either side.

## Documentation Map

[docs/SUMMARY.md](docs/SUMMARY.md) is the canonical mdBook table of contents — prefer it over guessing a
doc's path (a few README links elsewhere in this repo have gone stale before; `SUMMARY.md` is kept in
sync with the actual file tree). [docs/twinctl/twinctl.md](docs/twinctl/twinctl.md) documents `twinctl`'s
real subcommands and must be kept in sync whenever `ecotwin/scripts/twinctl.py`,
`ecotwin/info/`, `ecotwin/login/`, or `ecotwin/submission/` change their CLI surface.

## Agent Working Rules For This Repo

- Never commit. No `git commit`, no new branches, no branch switching. Leave changes uncommitted in the
  working tree; the user handles all git operations themselves.
- Read [CLAUDE.md](CLAUDE.md) first — it covers the Go/Python/Angular architecture, known dead code, and
  gotchas. Don't restate that content here.
- When changing `ecotwin/common/api_connector.py`'s backend integration, check and update the matching
  section of [repo/ianus-simod/agent_instructions.md](../ianus-simod/agent_instructions.md) in the same
  change — the two repos have no automated contract check between them.
- Don't remove the legacy `BackendConnector`/`AjaxApiHandler` (`/form/ajax`) path from
  `ecotwin/common/api_connector.py` without an explicit ask — it's deliberately kept working alongside the
  new token-based API, not superseded outright.
- Don't hand-edit `config/crds/`, `config/kcp/`, or `zz_generated.deepcopy.go` — regenerate via
  `make generate` (see [CLAUDE.md](CLAUDE.md) for the full codegen pipeline).
- Do not assume `ianus-simod` is checked out or importable from this repo — it's a separate sibling repo,
  consumed only over HTTP by `ecotwin/common/api_connector.py`.
