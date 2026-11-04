# Skills library

The Skills page opens the Library. Cards show the source, category, availability,
and remote update state. Search and filters work in both grid and list views;
opening a card retains the existing document preview and editor.
The secondary navigation stays visible, with a Library entry and the scoped
Skill list. Opening a card reveals and selects its entry in that list. Empty
groups are hidden while searching or filtering.

## Availability

- InkDock's built-in, user, and Project Skills are grouped separately from the
  shared library at `~/.agents/skills`.
- Shared Skills are discovered for management but excluded from Agent discovery,
  slash commands, Skill loading, and reference reads until explicitly enabled.
  Shared directories may be symlinks; supporting-file reads stay inside the
  selected Skill. InkDock never edits the shared library.
- Name precedence is Project, user, built-in, then shared. Disabling the winning
  Skill does not silently activate a lower-priority version. Agent profiles can
  further restrict availability, but cannot bypass the Library's disable switch.
- Global preferences are stored in `<data directory>/skills/.inkdock-library.json`;
  Project Skill availability is stored in `<project>/skills/.inkdock-library.json`.
  These contain scope/name keys, never absolute host paths. Preferences apply to
  Writing, Game, and other Skill-capable Agents.
- External engines receive the same host catalog and loader. Codex's automatic
  Skills instruction block is suppressed for InkDock-owned processes so its
  ambient library cannot bypass these preferences. InkDock does not rewrite the
  user's CLI Skills preferences.

## Genre cards and keyword routing

A Skill opts into genre routing by declaring the `genre-card` capability in its
front matter. Only enabled, Agent-available genre cards participate, so
installing an unrelated Skill never changes routing.

```yaml
---
name: genre-xianxia
description: 玄幻/仙侠升级流题材卡。
category: writing
capabilities:
  - genre-card
keywords:
  - 玄幻
  - 仙侠
  - 修仙
agent: general,ide
---
```

- `keywords` are matched against the current Writing message. Use
  genre-specific compound terms ("都市异能", "系统流"), not high-frequency
  words such as "都市" or "系统"; ASCII keywords match on word boundaries and
  must be at least three characters.
- When several cards match, the longest matched keyword wins, then the number of
  matches, then the Skill name. At most one card is preloaded per turn.
- Automatic loading applies only to Writing (`ide`) and general Agents, and is
  disabled in explicit-only catalogs. An explicit `/skill-name` in the same
  message suppresses the automatic duplicate.
- An automatic card is injected as a `turn.skill.keyword` fragment and labeled
  as an auto-loaded genre card in the runtime projection. If it would exceed the
  context budget, the turn continues without it. Explicit `/skill-name`
  invocations still fail closed when they do not fit.

## Import, export and remote updates

The Skills import picker uses the shared resource preview and install plan. It
accepts standard Skill repositories and ZIPs as well as mixed resource packages.
User and Project scopes remain explicit; shared and built-in Skills are never
modified in place. Individual Skills can be exported as a standard ZIP or added
to a mixed package.

Sources, content baselines and update preferences live in
`<data directory>/resource-exchange/installations/`. The Library links each
tracked Skill to its installation in Marketplace. That page provides manual
checks, reviewed updates, policy changes, detaching and backup recovery. It does
not replace the Skill editor or availability controls.

New installs default to manual updates. Notify checks daily without replacing
content. Automatic apply is available for packages containing only Skills,
presets and style references; it requires unchanged local files, the original
selection and dependencies, and idle consumers. A locally added file also counts
as a modification. Failed checks retain their attempt time, avoiding retry loops.
Mixed packages containing plugins, games or Project content require confirmation.

The installer freezes upstream bytes before confirmation and commits the whole
selected package with its source record. Before-images are retained in durable
transaction records. The market can download backups or review a restore plan;
it refuses to overwrite edits made since the original commit.

On upgrade from v0.5.0, `.inkdock-source.json` is backed up and moved into the shared
installation model. An explicitly enabled auto-update preference is retained;
other Skills remain manual. Skill content is unchanged. Downgrading requires
restoring the old source files from migration before-images to recover source
and update information. Existing legacy `.inkdock-backups` are not removed.
