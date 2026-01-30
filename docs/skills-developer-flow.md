# Best flow for developers: working with skills in the registry

Recommended paths for developers using the registry to **create**, **publish**, **discover**, and **use** skills.

---

## Choose your path

| Your situation | Best flow |
|----------------|-----------|
| **New skill from scratch** (you want to ship a skill) | [Flow A: CLI init → Docker publish](#flow-a-new-skill-cli-init--docker-publish) |
| **Existing skill in Git** (e.g. SKILL.md in a repo, no Docker) | [Flow B: Register via API or UI](#flow-b-existing-git-skill-register-via-api-or-ui) |
| **Curated catalog in Kubernetes** (operator-managed skills) | [Flow C: SkillCatalog CR](#flow-c-operator-curated-skillcatalog-cr) |
| **Find and use a skill** (discover, inspect, install) | [Flow D: Discover and use skills](#flow-d-discover-and-use-skills) |

---

## Flow A: New skill — CLI init + Docker publish

**Best for:** Building a new skill and publishing it so others can pull it via the registry.

1. **Scaffold** a skill project (SKILL.md + references):

   ```bash
   arctl skill init my-skill
   cd my-skill
   ```

2. **Edit** `SKILL.md` (frontmatter: `name`, `description`) and add your content/references.

3. **Publish** to the registry (builds a Docker image and registers the skill):

   ```bash
   arctl skill publish --docker-url <your-registry> ./my-skill
   # e.g. docker.io/myorg, ghcr.io/myorg
   ```

   Use `--push` to push the image; use `--tag` if you don’t want `latest`.

4. **Publish status** (if your registry creates skills as unpublished):

   ```bash
   arctl skill unpublish my-skill --version latest   # only if you need to unpublish
   # Publish is usually done by the publish command or admin UI/API
   ```

**Outcome:** Skill is in the registry with a Docker package. Others can run `arctl skill pull my-skill` to get the contents locally.

---

## Flow B: Existing Git skill — register via API or UI

**Best for:** Skills that live in Git only (e.g. [terraform-skill](https://github.com/antonbabenko/terraform-skill)): no Docker image, just SKILL.md + repo.

- **Option 1 — UI:** Registry → **Submit Resource** → choose **Skill**. Fill name, title, description, category, and (if applicable) repo/website URLs. Submit; then in **Registry** → Skills tab, open the skill and **Publish** so it appears under **Published**.
- **Option 2 — API:** `POST /v0/skills/publish` with a JSON body (name, version, description, repository, packages, remotes). Use `docs/sample-skill.json` as a template; leave `packages`/`remotes` empty for git-only. Then use admin API or UI to **publish** that version if it was created unpublished.

**Outcome:** Skill is discoverable in the registry (list/detail). Users install it from Git (e.g. `git clone` or Claude plugin marketplace), not via `arctl skill pull` (pull requires a Docker package).

---

## Flow C: Operator-curated — SkillCatalog CR

**Best for:** Platform/ops teams that want skills defined in GitOps and synced into the registry from the cluster.

1. **Define** a SkillCatalog manifest (see `config/samples/skillcatalog-terraform-skill.yaml`):

   ```yaml
   apiVersion: agentregistry.dev/v1alpha1
   kind: SkillCatalog
   metadata:
     name: my-skill
   spec:
     name: my-skill
     version: "1.0.0"
     title: My Skill
     category: infrastructure
     description: "..."
     websiteUrl: https://github.com/org/repo
     repository:
       url: https://github.com/org/repo
       source: github
   ```

2. **Apply** to the cluster (controller syncs catalog → registry):

   ```bash
   kubectl apply -f config/samples/skillcatalog-terraform-skill.yaml
   ```

3. **Verify:** `kubectl get skillcatalogs`; in the registry UI/API, the skill appears and can be published if the controller handles that.

**Outcome:** Skills are managed as Kubernetes resources and stay in sync with the registry; good for curated, org-wide catalogs.

---

## Flow D: Discover and use skills

**Best for:** Developers who want to find, inspect, and use skills from the registry.

1. **List** skills (CLI or UI):

   ```bash
   arctl skill list
   arctl skill list -o json   # or -o yaml
   ```

   In the UI: open **Registry** → **Skills** tab; use search/filters.

2. **Inspect** a skill (latest version):

   ```bash
   arctl skill show <skill-name>
   arctl skill show <skill-name> -o json
   ```

   In the UI: click a skill in the Skills tab to open the detail view.

3. **Pull** (only if the skill has a Docker package):

   ```bash
   arctl skill pull <skill-name>
   # Optional: arctl skill pull <skill-name> ./my-dir
   ```

   If the skill is git-only (no Docker package), get it from the repo/website shown in the registry (e.g. clone the repo or use the Claude plugin install command from the skill’s docs).

4. **Use** the skill in your environment (Claude Code, agent config, etc.) per that skill’s instructions; the registry is the place to discover and get metadata/links, not necessarily the only installer.

---

## Summary: one-line “best flow” by role

| Role | Best flow |
|------|-----------|
| **Developer shipping a new skill** | Flow A: `arctl skill init` → edit SKILL.md → `arctl skill publish --docker-url <registry>` |
| **Developer adding an existing Git skill to the registry** | Flow B: Submit Resource (UI) or `POST /v0/skills/publish` (API), then publish the version |
| **Operator curating skills for a team** | Flow C: Define SkillCatalog CRs, `kubectl apply`, controller syncs to registry |
| **Developer finding/using a skill** | Flow D: `arctl skill list` / UI → `arctl skill show <name>` → `arctl skill pull <name>` if Docker, else use repo/website from registry |

---

## Quick reference

| Goal | Command / action |
|------|-------------------|
| Create new skill project | `arctl skill init <name>` |
| Publish skill (Docker) | `arctl skill publish --docker-url <url> <path>` |
| List skills | `arctl skill list` or Registry → Skills |
| Show skill details | `arctl skill show <name>` |
| Pull skill (Docker package) | `arctl skill pull <name>` |
| Register git-only skill | UI Submit Resource → Skill, or `POST /v0/skills/publish` |
| Curate from cluster | SkillCatalog CR + `kubectl apply` |
| Unpublish | UI Published → Skills → Unpublish, or admin API |
| Delete version | `arctl skill delete <name> --version <ver>` |
