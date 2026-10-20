# PR Tag in build_image.yml Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `pr-tag` output (`0.0.0-PR-<number>`) to the `setup` job in `.github/workflows/build_image.yml` so PR builds get a semver-valid image tag instead of none.

**Architecture:** Single file change — add one job output, one step, and one line to the `tags` multiline input. No new files. Follows the exact pattern of the existing `tag` and `latest` outputs.

**Tech Stack:** GitHub Actions YAML

## Global Constraints

- Tag format must satisfy: `^[v]?(0|[1-9]\d*)(?:\.(0|[1-9]\d*))?(?:\.(0|[1-9]\d*))?(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
- Exact format: `0.0.0-PR-<pr-number>` (e.g. `0.0.0-PR-42`)
- Only fires on `pull_request_target` events
- Must not affect push-to-main or push-tag event behavior
- No changes to any file other than `.github/workflows/build_image.yml`

---

### Task 1: Add `pr-tag` output and step to `setup` job, pass to `build-image`

**Files:**
- Modify: `.github/workflows/build_image.yml:44-71`

**Interfaces:**
- Produces: `needs.setup.outputs.pr-tag` — string value `0.0.0-PR-<number>` or empty string

- [ ] **Step 1: Add the `pr-tag` output declaration to the `setup` job**

In `.github/workflows/build_image.yml`, find the `outputs:` block of the `setup` job (currently lines 44-46) and add the new output:

```yaml
    outputs:
      tag: ${{ steps.tag.outputs.tag || '' }}
      latest: ${{ steps.latest.outputs.latest || '' }}
      pr-tag: ${{ steps.pr-tag.outputs.pr-tag || '' }}
```

- [ ] **Step 2: Add the `pr-tag` step to the `setup` job**

After the existing `latest` step (currently line 58), add:

```yaml
      - id: pr-tag
        if: github.event_name == 'pull_request_target'
        run: echo "pr-tag=0.0.0-PR-${{ github.event.pull_request.number }}" >> $GITHUB_OUTPUT
```

- [ ] **Step 3: Add `pr-tag` to the `build-image` `tags` input**

In the `build-image` job, find the `tags:` multiline input (currently lines 69-71) and add the new output as a third line:

```yaml
      tags: |
        ${{ needs.setup.outputs.tag }}
        ${{ needs.setup.outputs.latest }}
        ${{ needs.setup.outputs.pr-tag }}
```

- [ ] **Step 4: Verify the final file looks correct**

The complete `setup` job outputs block and steps, and the `build-image` tags input, should read:

```yaml
  setup:
    needs: [gate]
    if: ${{ !cancelled() && (needs.gate.result == 'success' || needs.gate.result == 'skipped') }}
    permissions:
      contents: read
    runs-on: ubuntu-latest
    outputs:
      tag: ${{ steps.tag.outputs.tag || '' }}
      latest: ${{ steps.latest.outputs.latest || '' }}
      pr-tag: ${{ steps.pr-tag.outputs.pr-tag || '' }}
    steps:
      - name: Checkout code
        uses: actions/checkout@v6
        with:
          ref: ${{ github.event.pull_request.head.sha }}
          repository: ${{ github.event.pull_request.head.repo.full_name }}
      - id: tag
        if: github.event_name == 'push' && github.ref_type == 'tag'
        run: echo "tag=${{ github.ref_name }}" >> $GITHUB_OUTPUT
      - id: latest
        if: github.ref == format('refs/heads/{0}', github.event.repository.default_branch) && github.event_name == 'push'
        run: echo "latest=latest" >> $GITHUB_OUTPUT
      - id: pr-tag
        if: github.event_name == 'pull_request_target'
        run: echo "pr-tag=0.0.0-PR-${{ github.event.pull_request.number }}" >> $GITHUB_OUTPUT

  build-image:
    name: run-image-builder
    needs: [setup, gate]
    if: ${{ !cancelled() && needs.setup.result == 'success' && (needs.gate.result == 'success' || needs.gate.result == 'skipped') }}
    uses: kyma-project/test-infra/.github/workflows/image-builder.yml@main
    with:
      name: rt-bootstrapper
      dockerfile: Dockerfile
      context: .
      tags: |
        ${{ needs.setup.outputs.tag }}
        ${{ needs.setup.outputs.latest }}
        ${{ needs.setup.outputs.pr-tag }}
```

- [ ] **Step 5: Validate YAML syntax**

Run:
```bash
python3 -c "import yaml, sys; yaml.safe_load(open('.github/workflows/build_image.yml'))" && echo "YAML valid"
```
Expected output: `YAML valid`

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/build_image.yml
git commit -m "feat: add PR tag to build_image workflow"
```
