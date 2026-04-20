# Release Standard

Release policy for Helling artifacts and version progression.

## Versioning Inputs

- Product release version follows roadmap gates.
- API compatibility follows documented contract policy.

## Required Release Checks

- Generation and staleness gates pass.
- Lint and tests pass.
- Security baseline checks pass.
- Release notes/changelog updated.

## Hotfix Policy

- Hotfix branches are scoped to urgent defects only.
- Backport criteria must be explicit.
- Post-hotfix follow-up merges back to main are mandatory.
