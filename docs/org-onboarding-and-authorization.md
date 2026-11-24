# Org Onboarding & Authorization Workflow

How a brand-new person gets from "no account" to "provisioning provider services",
and the two separate systems (Keycloak authentication, OpenFGA authorization) that
both have to agree before any of it works. Written up after getting a live
`"You are not authorized to access this content"` on `/home/accounts` on this
local sandbox and root-causing it.

Status tags used below:
- **[VERIFIED]** — confirmed this session via live cluster logs/CRs or by reading
  the actual Platform Mesh operator code (not inferred).
- **[OBSERVED]** — the user saw this happen in the portal UI; not independently
  verified by reading code/logs.
- **[PROPOSAL]** — not implemented; an idea for how this *should* work later.
- **[UNTESTED]** — open question, not yet confirmed either way.

## Two separate identity/authorization planes

**[VERIFIED]** Authentication (Keycloak: "can this person log in?") and
authorization (OpenFGA: "is this person allowed to see this?") are independent
systems that both gate access. Logging in successfully proves nothing about
authorization — we hit exactly this: a real user could authenticate into an
org's realm but still got a 403 on `/accounts`, because no OpenFGA tuple existed
for them on that org's `Account` object.

**[VERIFIED]** Every org gets its own, separate Keycloak realm (e.g. `ecotwin-test`),
distinct from the shared `welcome` realm used for initial self-registration.
Logging into an org's realm is a second, independent login — not SSO carried
over from `welcome`. The org-realm user record for a given email is re-created
under that email, independently credentialed (own password, not synced from
`welcome`) — "copied" is close enough as shorthand, but there is no live link:
changing one password never changes the other.

## The workflow

1. **Create a user in the `welcome` realm.** Self-service registration. Only
   needed once per person; an existing `welcome` user skips straight to step 3.
2. **Log into `welcome`.** Registration auto-logs you in.
3. **Onboard a new organization.** Submitted from `/welcome`. This is **async**:
   it kicks off `account-operator` (creates an `Account` CR, type `org`) →
   `pm-keycloak-operator` (provisions the org's own Keycloak realm, then tries
   to auto-invite the creator into it) → `pm-openfga-operator` (writes an
   `owner` tuple on the org's `Account` object for the creator).
   **[VERIFIED]** The creator is *not* specified explicitly anywhere in this
   flow — `AccountSpec.Creator` is silently set to whoever is authenticated in
   `welcome` when the form is submitted. There is no "specify the owner" field
   in the onboarding UI we exercised.
   **[PROPOSAL]** Letting the onboarding form name an explicit owner (rather
   than always defaulting to the submitter) is a worthwhile future idea — not
   something the current code supports.
4. **Known local-sandbox gap: the invite step needs SMTP.** The auto-invite in
   step 3 sends an email with an activation link. This kind cluster has no SMTP
   server configured, so that send 500s every time, and the newly-created
   org-realm user ends up with **no password set at all** — a silent failure:
   the UI still reports "Organization successfully onboarded." Until this is
   fixed (real SMTP relay, or a direct Keycloak admin password-reset), a
   freshly onboarded org's owner may not be able to log into it at all on this
   sandbox. This is a Platform Mesh platform gap, not anything in this repo.
5. **Log into the new org's realm as its owner**, once step 4's gap is worked
   around. This is the second, independent login mentioned above.
6. **Provision services from inside the org.** Enabling a provider (injection
   molding, materials, generic materials) for the org is a separate, explicit
   action from logging in — not automatic just by being inside the org.

## Sub-accounts ("Accounts" section inside an org)

**[OBSERVED]** The user reported: inside an org, creating an entry under
"Accounts" and double-clicking it lets a "new user" provision services — raising
the question of what that user has that the org owner doesn't.

**[VERIFIED]** It is the same user, same identity — "new user" is a misnomer.
Evidence, read directly from the Platform Mesh source:

- `apis/core/v1alpha1/account_types.go`: there is exactly **one** `Account` CRD
  for both orgs and sub-accounts, distinguished only by `Spec.Type`
  (`AccountTypeOrg` vs `AccountTypeAccount`). Both are reconciled by the same
  `account-operator` `workspace` subroutine, which gives *every* Account —
  org or sub-account — its own nested kcp workspace. A sub-account is not a
  different kind of thing; it's the same CRD, nested one level deeper.
- Only `Type: "org"` additionally goes through `pm-keycloak-operator`'s
  realm-provisioning + invite flow. **No separate Keycloak user is ever created
  for a sub-account.**
- `pm-openfga-operator`'s generated authorization model
  (`operators/pm-openfga-operator/internal/subroutine/authorization_model.go`):
  every account-scoped object declares a `parent` relation, and:
  ```
  define member: [role#assignee] or owner or member from parent
  define owner:  [role#assignee] or owner from parent
  ```
  Ownership and membership **inherit down the parent chain automatically**.
  The org's owner is therefore already `owner` on every sub-account created
  under it, with no separate tuple ever written for them.

**Conclusion:** double-clicking a sub-account does not change who you're
authenticated as, and does not grant you anything you didn't already have as
org owner. It switches your *active workspace scope* into that sub-account's
own nested kcp workspace — the same relationship as `kubectl ws cd` into a
child workspace. Service enablement (APIBinding) then happens in that scope,
which is presumably why it reads as "provisioning services from a different
user" — same person, different workspace context.

**[UNTESTED]** Whether provisioning a service is scoped only to the
currently-active sub-account (so each sub-account enables providers
independently) or is somehow org-wide has not been confirmed either way and
needs a live walkthrough to settle.

## Open items

- **[PROPOSAL]** Explicit owner selection at org-onboarding time.
- **[UNTESTED]** Per-sub-account vs org-wide scope of service/provider
  enablement.
- **Platform gap, not ours to fix here:** invite-email delivery has no SMTP
  backing on this local sandbox, silently leaving new org-realm users
  password-less.
