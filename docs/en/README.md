# English documentation (`docs/en/`)

> **Source (Chinese SSOT)**: [`docs/README.md`](../README.md) — the Chinese docs landing page is the single
> source of truth (SSOT); this page is the English navigation adaptation, not a translation of it.
> **Source revision**: `dd78f36` (2026-09-29); adapted on 2026-09-30.

This directory holds the **English snapshots** of a documentation set whose authoritative language is
Chinese. It is deliberately small: only pages with independent value for English readers are translated.
Which docs get translated, in what priority, and how a snapshot is marked is defined by the coverage policy
in [`docs/i18n.md`](../i18n.md) §7 (中文).

## Start here

| I want to… | Read | Status |
|---|---|---|
| Understand the product / features / quick start | [`index.md`](index.md) | English translation snapshot of the root [`README.md`](../../README.md) |
| Understand the architecture and design decisions | [`architecture.md`](architecture.md) | English translation snapshot of [`docs/architecture.md`](../architecture.md) |
| Browse the complete documentation set | [`../README.md`](../README.md) | Chinese; the human navigation SSOT (`docs/`) |
| Know what is translated and why | [`../i18n.md`](../i18n.md) | Chinese; coverage policy §7 |
| Read the API/SDK surface as machine-readable data | [`../api/openapi.json`](../api/openapi.json) · [`../api/accounts.schema.json`](../api/accounts.schema.json) | English-friendly; no translation needed |

## Chinese-only docs (high value)

Everything below exists **in Chinese only** right now. The links are given so English readers can reach the
authoritative pages (browser translation works reasonably well on them); they are not translation promises.

| I want to… | Chinese doc |
|---|---|
| Install / configure / operate the service | [`../DEPLOYMENT.md`](../DEPLOYMENT.md) · [`../OPERATIONS.md`](../OPERATIONS.md) |
| Look up a configuration variable | [`../CONFIGURATION.md`](../CONFIGURATION.md) (SSOT for all `S3C_*` variables) |
| Call the REST API or map errors | [`../api.md`](../api.md) · [`../errors.md`](../errors.md) |
| Read the user-facing UI guide | [`../user-guide.md`](../user-guide.md) |
| Look up S3 / project terminology | [`../glossary.md`](../glossary.md) |
| Check versioning, support windows, browser/OS matrix | [`../compatibility.md`](../compatibility.md) |
| Review the security boundary / threat model | [`../threat-model.md`](../threat-model.md) |
| See accessibility status and limitations | [`../accessibility.md`](../accessibility.md) |
| Contribute code (TDD-first rules, gates) | [`../DEVELOPMENT.md`](../DEVELOPMENT.md) |
| Use an AI coding agent safely | [`../AI_POLICY.md`](../AI_POLICY.md) · [`../AGENT_EVALS.md`](../AGENT_EVALS.md) |
| Record / review architecture decisions | [`../decisions/index.md`](../decisions/index.md) |
| Track features, known issues, roadmap | [`../FEATURES.md`](../FEATURES.md) · [`../KNOWN_ISSUES.md`](../KNOWN_ISSUES.md) · [`../ROADMAP.md`](../ROADMAP.md) |
| See release history | [`../../CHANGELOG.md`](../../CHANGELOG.md) |
| Report a vulnerability / ask a question / read governance | [`../../.github/SECURITY.md`](../../.github/SECURITY.md) · [`../../.github/SUPPORT.md`](../../.github/SUPPORT.md) · [`../../.github/GOVERNANCE.md`](../../.github/GOVERNANCE.md) · [`../../.github/CODE_OF_CONDUCT.md`](../../.github/CODE_OF_CONDUCT.md) |
| Read frozen historical snapshots | [`../archive/index.md`](../archive/index.md) |

## Maintaining this directory

- **Chinese is the SSOT.** Translate facts only: never add, drop or "fix" content. Code, paths, metric
  names, environment variables, SDK method names and ADR link targets stay verbatim.
- **Every page except `index.md` declares its source.** Put a `**Source (Chinese SSOT)**:` line with a
  relative link to the Chinese source and a `**Source revision**:` line (short commit hash + date) at the
  top. `apps/server/en_docs_gate_test.go` red-lights a page that is missing either declaration, whose source
  does not exist, or whose source is not under `docs/`.
- **Every page is reachable.** New pages must be linked from this file (or from
  [`../README.md`](../README.md)); `doc_index_gate_test.go` and `en_docs_gate_test.go` enforce it.
- **`index.md` tracks the root README.** Its version literal and its "N `/api/*` endpoints" claim are
  compared mechanically against the root [`README.md`](../../README.md).
- **Run the gate after touching this directory**:

  ```bash
  cd apps/server && go test . -run 'TestEnDocs' -count=1
  ```

- **Review cadence is a recommendation, not CI-enforced** (see [`../i18n.md`](../i18n.md) §7.4): translate
  in the same PR as the Chinese change; otherwise re-review at the interval listed in
  [`../DEVELOPMENT.md`](../DEVELOPMENT.md) §4's doc registry.
