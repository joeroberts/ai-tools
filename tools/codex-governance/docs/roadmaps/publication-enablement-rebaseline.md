# Publication Enablement and Rebaseline Delivery Roadmap

## Phase 1: Enable Signed Publication

Complete both components in one bounded implementation slice.

### Component 1: Issue Exact Owner Authorization

Add explicit repository-owner signer bootstrap and version-2 authorization
issuance with owner-only files, policy trust, exact run and remote bindings,
bounded expiry, no overwrite, and no remote side effect.

### Component 2: Separate Target Base and Prove Lineage

Validate implementation and remote-target SHAs independently, require both to
be ancestors of the authorized commit, preserve version-1 behavior, and reject
remote target movement before publication.

## Delivery Order

After completing and validating the bounded slice, use the resulting local
binary to issue the exact signed authorization needed to publish the branch
containing `REK-28` without changing its reviewed diff.
