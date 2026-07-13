# Implementation-Run Fixture Catalog

These fixtures are defined in Phase 1. Phase 2 supplies the
`implementation-run` schema and turns this catalog into executable fixtures.
Fixtures must contain only synthetic source, ticket, model, Git, and remote
identifiers.

| Fixture ID | Scenario | Expected disposition |
| --- | --- | --- |
| `valid-local-run` | Approved work item, fresh source, bounded task bundle, valid scoped diff, and passing checks | reaches `ready-to-commit` |
| `source-drift` | Ticket digest or acceptance criteria differs from the baseline | `escalated`; no dispatch |
| `scope-violation` | Adapter result changes an unapproved path or exceeds budget | `escalated`; no commit |
| `adapter-crash-recovery` | Restart during `running` with a known fake-adapter task ID | reconciles without redispatch |
| `unknown-adapter-task` | Restart during `running` with no provider task | `escalated`; no redispatch |
| `review-remediation` | Named blocking finding is fixed in an approved path | returns to review within two cycles |
| `remote-authorization-mismatch` | Authorization SHA, branch, remote, operation, or expiry differs | remote action rejected |
| `protected-branch` | Valid authorization names a protected or default branch | remote action rejected |

Phase 2 may add JSON fixtures beneath this directory only after the schema and
field-version contract are approved. Do not include credentials, real remote
URLs, local worktree paths, raw prompts, or unrestricted logs.
