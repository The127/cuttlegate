# ADR 0029: localStorage client persistence policy

**Date:** 2026-03-23
**Status:** Accepted
**Issue:** #271

## Context

Sprint 20 (#271 — first-run onboarding flow) introduced the first use of `localStorage` in the Cuttlegate SPA. Two keys were written:

- `cg:sdk_tab` — the last-selected SDK tab (Go/JS/Python) in the onboarding prompt
- `cg:sdk_prompt_dismissed` — whether the user has permanently dismissed the SDK setup prompt

Prior to this sprint, the SPA had no client-side persistence outside of the OIDC session (managed by the OIDC library). The absence of a documented policy means the next developer to reach for `localStorage` has no guidance on what belongs there, how keys should be named, or what must never be stored.

**The risk without a policy:** auth tokens, API keys, or other secrets end up in `localStorage` because "we already use it." The pattern must be constrained before it proliferates.

## Decision

`localStorage` is permitted in the Cuttlegate SPA for **UI preference state only** — values whose loss or corruption has no security consequence and no data integrity consequence.

### What may be stored in localStorage

- UI preferences: last-selected tab, collapsed/expanded panel state, dismissed-once UI hints
- Non-sensitive onboarding state: whether a one-time prompt has been shown

### What must never be stored in localStorage

- Auth tokens, session tokens, OIDC tokens of any kind
- API keys or any credential
- User-identifying data (email, user ID, project membership)
- Any value that, if read by injected script, would grant access to a resource

OIDC session state is managed by the existing OIDC library's storage strategy — that library's storage choice is outside the scope of this ADR and must not be changed without a dedicated decision.

### Key naming convention

All localStorage keys written by Cuttlegate must use the `cg:` prefix followed by a dot-separated path:

```
cg:<feature>.<attribute>
```

Examples:
- `cg:sdk_tab` — acceptable (Sprint 20 existing key; grandfathered)
- `cg:sdk_prompt_dismissed` — acceptable (Sprint 20 existing key; grandfathered)
- Future keys must follow `cg:<feature>.<attribute>` strictly, e.g. `cg:flags.sort_order`

The `cg:` prefix prevents collisions with third-party scripts and makes Cuttlegate keys identifiable in DevTools.

### No abstraction layer required

Direct `localStorage.getItem` / `localStorage.setItem` calls are acceptable. A wrapper abstraction is not required and should not be introduced speculatively. If the number of keys grows beyond ~10, revisit.

### Reset contract

localStorage state is user-controlled. There is no server-side reset mechanism. The UI must behave correctly when `localStorage` is empty (first load, private browsing, cleared storage) — stored values are always hints, never requirements.

## Rationale

**Preference state only keeps the risk surface small.** If a user's tab selection is lost, nothing breaks. If an auth token in localStorage is read by injected script, the user's account is compromised. The distinction is categorical, not a matter of degree.

**No abstraction layer now.** Two keys do not justify an abstraction. The policy is the constraint; the abstraction is a future option if volume warrants it.

**Key naming via prefix.** The `cg:` prefix is the simplest collision-avoidance strategy that costs nothing and gives operators clear visibility in DevTools.

## Consequences

- Every new `localStorage` usage must be reviewed against this ADR at grooming. If the proposed value does not meet the "UI preference only" criterion, it does not go in `localStorage`.
- The two Sprint 20 keys (`cg:sdk_tab`, `cg:sdk_prompt_dismissed`) are grandfathered — they comply with the policy.
- Auth flows must never use `localStorage` for token storage. This is a hard rule; any proposal to do so requires a new ADR and explicit security sign-off.
- New keys added in future issues must use the `cg:<feature>.<attribute>` naming pattern.
