# AGENTS.md

This file provides guidance to Claude Code when working with the docs module.

---

## Role

Documentation for doc-thor itself. This folder is structured as a standard MkDocs project
and is intended to be built **using doc-thor** once the tool is functional. That's the point —
doc-thor eats its own output. Until it can do that, these docs can be built manually with
a local `mkdocs serve`.

---

## Content Structure

```
docs/
├── mkdocs.yml                           # MkDocs project config
├── index.md                             # What doc-thor is and what it does
├── architecture.md                      # System architecture and module map
└── philosophy/                          # Opinion pieces on documentation practices
    ├── centralized-documentation.md
    ├── docs-close-to-code.md
    ├── git-versioned-docs.md
    └── technical-vs-commercial.md
```

---

## Tone

The philosophy docs are written with a direct, slightly sarcastic voice. They make a point
without hedging, and they don't pad sentences to sound polite. If you add content to that
section, match the tone. The rest of the docs (index, architecture) are straightforward and
neutral.

---

## Building Locally

```bash
pip install mkdocs
cd docs
mkdocs serve        # Development server at http://localhost:8000
mkdocs build        # Produces a /site directory with the static output
```

---

## Eventually

Once doc-thor is running, register this `docs/` folder as a project and let the pipeline
handle it. The `mkdocs.yml` here is the build config. The pipeline will clone the repo,
run `mkdocs build`, upload to storage, and serve it at a subdomain.

---

<skills_system priority="1">

## Available Skills

<!-- SKILLS_TABLE_START -->
<usage>
When users ask you to perform tasks, check if any of the available skills below can help complete the task more effectively. Skills provide specialized capabilities and domain knowledge.

How to use skills:

- Invoke: Bash("openskills read <skill-name>")
- The skill content will load with detailed instructions on how to complete the task
- Base directory provided in output for resolving bundled resources (references/, scripts/, assets/)

Usage notes:

- Only use skills listed in <available_skills> below
- Do not invoke a skill that is already loaded in your context
- Each skill invocation is stateless
  </usage>

<available_skills>

<skill>
<name>devops-architect</name>
<description>></description>
<location>project</location>
</skill>

<skill>
<name>mermaid-diagram</name>
<description>Generate well-structured Mermaid diagrams for visual documentation. Use when designing systems, documenting workflows, creating architecture diagrams, or any task requiring visual representation of processes, relationships, or structures. Triggers include requests like "create a diagram", "visualize this", "show the flow", "draw the architecture", or when explaining complex systems that benefit from visual aids.</description>
<location>project</location>
</skill>

<skill>
<name>skill-creator</name>
<description>Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.</description>
<location>project</location>
</skill>

</available_skills>

<!-- SKILLS_TABLE_END -->

</skills_system>
