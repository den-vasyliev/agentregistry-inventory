# User Experience: Working with Skills from the Registry

## Summary

Skills are surfaced in **CLI**, **UI**, and **API** with parity to servers and agents in most flows. The main friction is **publish/pull** being Docker-centric while many skills (e.g. Claude/terraform-skill) are git-based; **SkillCatalog (Kubernetes)** is not yet reflected in the UI/CLI.

---

## What Exists

### CLI (`arctl skill`)

| Command | Purpose | Notes |
|--------|---------|--------|
| `list` | List skills from registry | Pagination, table/json/yaml, `--all`, `--page-size` |
| `show <name>` | Show latest version of a skill | Table or JSON (`-o json`); JSON currently broken (prints Go struct) |
| `init [name]` | Scaffold new skill project | SKILL.md + references; suggests `publish --docker-url` |
| `publish <path>` | Publish skill | **Docker-only**: builds image from SKILL.md folder, pushes to registry; requires `--docker-url`, optional `--push`, `--tag`, `--platform` |
| `pull <name> [dir]` | Pull skill locally | **Docker-only**: requires skill to have a `docker` package; extracts image to dir (default `./skills/<name>`) |
| `unpublish` | Unpublish a version | Admin; changes visibility |
| `delete <name> --version` | Delete a version | Checks published state unless `--force` |
| `remove <name>` | Remove skill | **Not implemented** (no-op) |

### UI

- **Registry (home)**: Tabs for Servers, Agents, **Skills**. List + detail (SkillCard, SkillDetail), search, import (Import Skills), add (Add Skill), publish from list.
- **Published**: Tab “Skills” alongside MCP Servers and Agents. List published skills, unpublish; **no Deploy** for skills (deploy is server/agent only).
- **Deployed**: Deployments are server/agent; skills are not deployable as runtime targets.
- **Submit Resource**: Can submit a “Skill” (name, title, description, category); flows into registry.

### API

- Public: `GET /v0/skills`, `GET /v0/skills/{name}/versions/{version|latest}`, `GET /v0/skills/{name}/versions`, `POST /v0/skills/publish` (create/update).
- Admin: create at `POST /admin/v0/skills`, publish/unpublish at `.../skills/{name}/versions/{version}/publish` and `.../unpublish`.
- List supports `cursor`, `limit`, `updated_since`, `search`, `version=latest`.

### Kubernetes (SkillCatalog)

- **SkillCatalog** CR and controller sync catalog → registry (or discovery). Sample: `config/samples/skillcatalog-terraform-skill.yaml`.
- No CLI/UI yet to list or manage SkillCatalogs; users use `kubectl get skillcatalogs` / `kubectl apply`.

---

## Strengths

1. **Consistent surface**: Skills share list/detail/publish/unpublish patterns with servers and agents in UI and API.
2. **Discovery**: Search and filters (including “Skills” tab) make it easy to see what’s published.
3. **Admin flows**: Publish/unpublish/delete with guards (e.g. unpublish before delete unless `--force`).
4. **Init**: `arctl skill init` gives a clear starting point and hints for publish.

---

## Gaps & Friction

1. **Docker-only publish/pull**
   - Many skills (e.g. [terraform-skill](https://github.com/antonbabenko/terraform-skill)) are git + SKILL.md, no Docker.
   - `arctl skill publish` always builds a Docker image; `pull` only works when a skill has a `docker` package. So git-only skills can be registered (e.g. via API or Submit Resource) but not published or pulled via CLI in the same way.
   - **Suggestion**: Support “register from Git URL” or manifest (no Docker), and/or `pull` that clones a repo when no Docker package is present.

2. **`arctl skill show -o json`**
   - Prints Go struct, not JSON. **Fix**: Use encoder (e.g. `json.Marshal` + `os.Stdout`) like `list` does.

3. **`arctl skill remove`**
   - Implemented as no-op (“Not implemented yet”). Either implement (e.g. delete from registry or local only) or remove/hide and point users to `delete` or API.

4. **Skills not deployable**
   - Published page has “Deploy” for servers/agents only. Skills are not first-class deployment targets (by design if they’re metadata/capability descriptors). If “install skill to environment” becomes a product flow, it could be a separate action (e.g. “Add to agent” or “Install to Claude”) rather than “Deploy” like MCP.

5. **SkillCatalog not in UI/CLI**
   - Operators manage SkillCatalogs with `kubectl`. No registry UI or `arctl` commands to list/create SkillCatalogs or map “this SkillCatalog → this registry skill.” Optional: read-only list of SkillCatalogs in UI, or `arctl skill catalog list` that talks to the cluster.

6. **Version in list**
   - CLI list shows one row per skill (from `GetSkills()`). If the API returns only latest per name, that’s clear; if it can return multiple versions, the table might need a “Version” column (it already has Version in the headers—confirm behavior matches).

7. **Docs**
   - No single “Skills” user doc: how to register (API vs CLI vs UI vs SkillCatalog), when to use Docker vs git, how to pull/install. A short “Using skills” doc would help.

---

## Recommendations (prioritized)

1. **Fix** `arctl skill show -o json` to output valid JSON.
2. **Clarify or implement** `arctl skill remove` (implement vs remove command vs document as “local only”).
3. **Document** recommended flows: git-only skill (API/Submit/SkillCatalog) vs Docker skill (init → publish → pull).
4. **Consider** “register from Git” and/or “pull = clone” for skills without a Docker package to improve UX for terraform-skill-style skills.
5. **Optional**: Expose SkillCatalogs in UI or CLI for operators who manage skills via Kubernetes.

---

## Quick reference: “I want to…”

| Goal | Current UX |
|------|------------|
| List skills | UI: Registry → Skills tab. CLI: `arctl skill list`. |
| See one skill | UI: Click skill → detail. CLI: `arctl skill show <name>`. |
| Publish a Docker-wrapped skill | CLI: `arctl skill publish --docker-url <url> ./path` (needs SKILL.md). |
| Publish a git-only skill | API `POST /v0/skills/publish` with JSON, or UI Submit Resource → Skill, or apply a SkillCatalog CR. |
| Pull a skill | CLI: `arctl skill pull <name>` (only if skill has Docker package). |
| Unpublish | UI: Published → Skills → Unpublish. CLI: `arctl skill unpublish <name> --version <ver>`. |
| Delete a version | CLI: `arctl skill delete <name> --version <ver>`. |
| Manage via Kubernetes | `kubectl apply -f config/samples/skillcatalog-terraform-skill.yaml`, `kubectl get skillcatalogs`. |
