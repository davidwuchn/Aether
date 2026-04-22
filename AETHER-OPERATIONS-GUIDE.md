# **Aether Operations Guide**

*Printable operator sheet for developing, testing, publishing, and updating Aether safely.*

## **1. The One-Sentence Model**

**`aether` is the public/stable runtime. `aether-dev` is the isolated development runtime.**

The two things to remember are:

- **Global runtime state** lives in a **hub** under your home directory.
- **Repo colony state** lives inside each repo under **`.aether/`**.

> **Changing the channel changes the global runtime.**  
> **Changing the repo changes the local colony state.**

---

## **2. What Lives Where**

```mermaid
flowchart TD
    A[Source repo<br/>/Users/callumcowie/repos/Aether] --> B[Install / publish step]

    B --> C1[Stable hub<br/>~/.aether]
    B --> C2[Dev hub<br/>~/.aether-dev]

    C1 --> D1[Stable binary<br/>aether]
    C2 --> D2[Dev binary<br/>aether-dev]

    D1 --> E1[Stable test repo<br/>repo/.aether/data]
    D2 --> E2[Dev test repo<br/>repo/.aether/data]
```

### **Global machine state**

**Stable**
- **binary:** `aether`
- **hub:** `~/.aether`

**Dev**
- **binary:** `aether-dev`
- **hub:** `~/.aether-dev`

### **Repo-local colony state**

Inside *every* target repo:

- `repo/.aether/data/`
- `repo/.aether/CONTEXT.md`
- `repo/.aether/HANDOFF.md`
- planning, survey, build, continue artifacts

### **Critical consequence**

**`aether-dev` gives you a clean global Aether. It does *not* automatically give you a clean repo colony.**

So:

- if you want to test **unreleased runtime changes**, `aether-dev` is enough for the **global** side
- if you want a **truly clean dev test**, use a **separate repo copy or worktree** as well

---

## **3. Stable vs Dev**

```mermaid
flowchart LR
    S1[aether] --> S2[~/.aether]
    S2 --> S3[Public / stable repos]

    D1[aether-dev] --> D2[~/.aether-dev]
    D2 --> D3[Dev test repos / worktrees]
```

### **Use `aether` when**

- you want **public/stable behavior**
- you want to test what a real user should get
- you are using the **published** install/update path

### **Use `aether-dev` when**

- you changed the Aether source repo locally
- you want to test **unreleased** runtime or wrapper changes
- you do **not** want to overwrite the public/stable runtime on this machine

---

## **4. Best Practice**

This is the **correct operating model**:

1. **One codebase**
   - `/Users/callumcowie/repos/Aether`

2. **Two channels**
   - `aether` + `~/.aether`
   - `aether-dev` + `~/.aether-dev`

3. **Two kinds of test repo**
   - **stable repo/worktree**
   - **dev repo/worktree**

4. **npm stays stable-only**
   - `npx --yes aether-colony@latest` is for the **public** channel

5. **GitHub Releases stay stable-only**
   - release assets are the source of truth for the public runtime

### **Do not do this**

Do **not** compare stable and dev by flipping the **same repo copy** back and forth between:

```bash
aether update --force
aether-dev update --force
```

That works technically, but it is easy to confuse because the repo-local `.aether/data` remains the same colony history.

---

## **5. Daily Aether Development Workflow**

Use this when you are actively changing Aether itself.

```mermaid
flowchart TD
    A[Edit Aether source] --> B[Run tests]
    B --> C[Publish dev channel from source]
    C --> D[Update dev test repo]
    D --> E[Restart session]
    E --> F[Test actual behavior]
```

### **Step A — Work in the source repo**

```bash
cd /Users/callumcowie/repos/Aether
```

### **Step B — Run tests**

```bash
go test ./... -count=1
```

### **Step C — Publish the dev channel from source**

```bash
go run ./cmd/aether install --channel dev --package-dir "$PWD" --binary-dest "/Users/callumcowie/repos/Aether-dev/bin"
```

This does **all** of the following:

- refreshes the **dev hub** at `~/.aether-dev/`
- rebuilds the **dev binary** as `aether-dev`
- leaves the public/stable runtime alone

### **Step D — Update a dev test repo**

In the repo you want to test:

```bash
aether-dev update --force
```

### **Step E — Restart the session**

After install or update:

- close the current Claude/OpenCode/Codex session
- open a **fresh** session in that repo

**Why:** file changes on disk do not automatically refresh a live tool session.

---

## **6. Stable / Public Testing Workflow**

Use this when you want to test what an actual user gets.

### **If the repo already uses Aether**

```bash
aether update --force --download-binary
```

### **If the repo has never been set up**

```bash
aether lay-eggs
aether update --force --download-binary
```

### **Then**

- close the current session
- open a fresh session in that repo

### **Rule**

Use a **stable repo copy/worktree** for this.  
Do **not** use your dev test repo for public/stable verification.

---

## **7. Exact Release Workflow**

```mermaid
flowchart TD
    A[Change source] --> B[Set release version]
    B --> C[Commit and push main]
    C --> D[Push annotated tag]
    D --> E[GitHub Release publishes Go binaries]
    E --> F[npm publishes aether-colony]
    F --> G[Users install stable runtime]
```

## **Release variables**

Choose the release version first:

```bash
VERSION="1.0.20"
```

## **Step 1 — Update the version in both places**

Edit these files so they both match **exactly**:

- `.aether/version.json`
- `npm/package.json`

## **Step 2 — Commit and push**

```bash
git add .aether/version.json npm/package.json
git commit -m "release: prepare v$VERSION"
git push origin main
```

## **Step 3 — Push the annotated release tag**

```bash
git tag -a "v$VERSION" -m "v$VERSION"
git push origin "v$VERSION"
```

## **Step 4 — Verify the GitHub release**

```bash
gh release view "v$VERSION"
```

If the release workflow did **not** trigger automatically, try:

```bash
gh workflow run Release -f tag="v$VERSION"
```

If GitHub still refuses to run it, use the documented manual fallback:

```bash
export GITHUB_TOKEN="$(gh auth token)"
goreleaser release --clean
```

## **Step 5 — Publish npm only after GitHub release assets exist**

First authenticate if needed:

```bash
npm login --registry=https://registry.npmjs.org/
npm whoami
```

Then publish:

```bash
cd /Users/callumcowie/repos/Aether/npm
npm publish --access public
```

## **Step 6 — Verify npm**

```bash
npm view aether-colony dist-tags --json
npm view aether-colony version
```

Expected:

- `latest` points to the same version as GitHub release `v$VERSION`
- package version matches `.aether/version.json`

---

## **8. Public Install Paths**

### **Go install**

```bash
go install github.com/calcosmic/Aether@latest
aether install
```

### **npm bootstrap**

```bash
npx --yes aether-colony@latest
```

### **Important**

**npm is a bootstrap/discovery path, not a second runtime.**  
It should always point to the same **stable** public release version.

---

## **9. Exact Verification Commands**

## **Source repo verification**

```bash
cd /Users/callumcowie/repos/Aether
go test ./... -count=1
go build ./cmd/aether
```

## **Stable hub verification**

```bash
find ~/.aether/system/commands/claude -maxdepth 1 -type f | wc -l
find ~/.aether/system/commands/opencode -maxdepth 1 -type f | wc -l
find ~/.aether/system/agents -maxdepth 1 -type f | wc -l
find ~/.aether/system/codex -maxdepth 1 -type f | wc -l
find ~/.aether/system/skills-codex -name SKILL.md | wc -l
```

## **Dev hub verification**

```bash
find ~/.aether-dev/system/commands/claude -maxdepth 1 -type f | wc -l
find ~/.aether-dev/system/commands/opencode -maxdepth 1 -type f | wc -l
find ~/.aether-dev/system/agents -maxdepth 1 -type f | wc -l
find ~/.aether-dev/system/codex -maxdepth 1 -type f | wc -l
find ~/.aether-dev/system/skills-codex -name SKILL.md | wc -l
```

Expected counts:

- **Claude commands:** `50`
- **OpenCode commands:** `50`
- **OpenCode agents:** `25`
- **Codex agents:** `25`
- **Codex skills:** `29`

---

## **10. Safe Testing Matrix**

| **What you are testing** | **Binary** | **Hub** | **Repo** |
|---|---|---|---|
| Public stable behavior | `aether` | `~/.aether` | stable repo/worktree |
| Unreleased source changes | `aether-dev` | `~/.aether-dev` | dev repo/worktree |
| Side-by-side comparison | both | both | two separate repo copies/worktrees |

---

## **11. Do / Don’t**

### **Do**

- **Do** use `aether-dev` for unreleased Aether development
- **Do** use `aether` for public/stable testing
- **Do** use separate repo copies/worktrees when comparing stable vs dev
- **Do** restart sessions after `install` or `update`
- **Do** keep `.aether/version.json` and `npm/package.json` identical for stable releases

### **Don’t**

- **Don’t** use npm as a second runtime
- **Don’t** compare stable vs dev in the same repo if you want a clean result
- **Don’t** assume changed files on disk means the live session is refreshed
- **Don’t** let source-development overwrite the stable/public runtime

---

## **12. Short Version**

```text
aether      = public / stable
aether-dev  = private development runtime

~/.aether      = stable global brain
~/.aether-dev  = dev global brain

repo/.aether/data = local colony state for that repo only
```

## **The two commands you will use most**

### **When developing Aether itself**

```bash
cd /Users/callumcowie/repos/Aether
go run ./cmd/aether install --channel dev --package-dir "$PWD" --binary-dest "/Users/callumcowie/repos/Aether-dev/bin"
```

### **When testing those changes in a target repo**

```bash
aether-dev update --force
```
