# Skill Marketplace

Install skills from GitHub repositories.

## Install a Skill

```bash
roland skill install https://github.com/user/debugging-skill
roland skill install https://github.com/user/debugging-skill --name debug
```

What happens:
1. Clones the repo (shallow, `--depth=1`)
2. Validates `SKILL.md` exists
3. Copies to `~/.roland/skills/<name>/` (`.git` excluded)
4. Registers in `skills.json` with source URL and version (git tag or commit hash)

## Uninstall

```bash
roland skill remove <name>
```

Removes the skill directory and deregisters from `skills.json`.

## Skill Format

A skill is a directory containing:
- `SKILL.md` (required) — the skill content injected into agent context
- Optional: `skill.json` with metadata (personas, task types, tags for auto-matching)

Skills are **markdown + config only** — they do NOT contain executable code.

## Version Tracking

Each installed skill tracks:
- `source`: the git URL it was installed from
- `version`: git tag (e.g., `v1.2.0`) or short commit hash

View with `roland skill list`.
