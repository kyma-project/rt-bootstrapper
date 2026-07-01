---
name: pr-tag-build-image
description: Add a semver-compatible PR tag to the build_image.yml workflow as an alternative to the release tag
metadata:
  type: project
---

# PR Tag in build_image.yml

## Problem

The image-builder validates image tags against a semver pattern:
`^[v]?(0|[1-9]\d*)(?:\.(0|[1-9]\d*))?(?:\.(0|[1-9]\d*))?(?:-(...))?(?:\+(...))?$`

PR builds currently produce no version tag (`tag` output is empty on `pull_request_target` events), making PR images harder to identify and reference.

## Solution

Add a `pr-tag` output to the `setup` job that emits `0.0.0-PR-<number>` when the workflow runs on a `pull_request_target` event. This tag is mutually exclusive with the semver release tag.

## Design

### `setup` job

Add one output and one step:

```yaml
outputs:
  tag: ${{ steps.tag.outputs.tag || '' }}
  latest: ${{ steps.latest.outputs.latest || '' }}
  pr-tag: ${{ steps.pr-tag.outputs.pr-tag || '' }}   # new

steps:
  # ... existing steps ...
  - id: pr-tag
    if: github.event_name == 'pull_request_target'
    run: echo "pr-tag=0.0.0-PR-${{ github.event.pull_request.number }}" >> $GITHUB_OUTPUT
```

### `build-image` job

Add the new output to the `tags` multiline input:

```yaml
tags: |
  ${{ needs.setup.outputs.tag }}
  ${{ needs.setup.outputs.latest }}
  ${{ needs.setup.outputs.pr-tag }}
```

## Tag Behavior Per Event

| Event | `tag` | `latest` | `pr-tag` |
|---|---|---|---|
| `push` + tag ref | `v1.2.3` | — | — |
| `push` to main | — | `latest` | — |
| `pull_request_target` | — | — | `0.0.0-PR-42` |

Empty outputs produce blank lines in the multiline `tags` input, which image-builder already handles safely (consistent with existing behavior).

## Out of Scope

- No changes to `release.yml` or any other workflow.
- No renaming of existing outputs.
