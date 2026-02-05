# Why doc-thor

## Your current setup is broken. You just don't know it yet.

Somewhere in your organization, documentation is living in a Google Doc nobody
bookmarked, a wiki nobody updates, and a README that was accurate in 2022. You are
either aware of this and quietly ignoring it, or you are about to find out the hard way.

doc-thor doesn't fix your writing. It fixes the pipeline that should be carrying your
writing to your users — the one that, in most setups, doesn't exist.

---

## Self-hosted. No strings attached.

No SaaS. No monthly bill. No "contact sales for the features you actually need."
You run it. You own it. You can air-gap it if your security team has opinions.

One Docker Compose file. One command. The whole stack, on your infrastructure,
under your control. Nobody is phoning home. Nobody is charging you per-seat.
It's just software, doing its job, on your servers.

---

## Every version lives forever. Automatically.

You ship v2.1. The docs for v2.1 get built, uploaded, and served at their own URL.
Nobody has to remember to do this. Nobody has to "freeze" the docs before release.
It just happens, because that's what the pipeline does.

A developer running v2.1 six months from now reads the v2.1 docs. Not "the latest."
Not "close enough." The exact docs that matched their version. This is how it should
have always worked.

---

## The build pipeline doesn't care what you use.

MkDocs? Fine. Sphinx? Sure. Hugo? Sure. A shell script that assembles HTML from
fragments? As long as it runs in a container and writes to `/output`, doc-thor will
build it, upload it, and serve it.

No plugin ecosystem to navigate. No compatibility matrix. One contract: your image
produces files, the pipeline ships them. That's the whole agreement.

---

## Horizontal scaling is a number in a config file.

Builds are slow? Builds are piling up? Add another builder instance. They
coordinate automatically. The server hands out jobs one at a time, atomically.
No duplicates, no leader election, no coordination protocol beyond changing
a single number:

```yaml
deploy:
  replicas: 3   # that's it
```

---

## Subdomains configure themselves.

Register a project. Publish a version. The subdomain exists. You didn't write
an Nginx config. You didn't restart anything. The system noticed, updated itself,
and started serving traffic.

The only manual step is the one that should be manual: deciding what to publish.
Everything else is handled.

---

## Git does the heavy lifting. doc-thor does the rest.

Docs live in your repo. They version with your code, branch with your branches,
and get reviewed in your PRs. doc-thor picks up from there: builds, publishes,
serves. The gap between "code is shipped" and "docs are live" is closed.
Not reduced. Closed.

---

## It dogfoods itself.

These docs — the ones you're reading right now — are built and served by doc-thor.
If the tool breaks, its own documentation breaks first. There is no better
incentive structure for reliability than that.

---

## The short version

- Free. Open. On-premise. No paywall, no SaaS, no vendor lock-in.
- Automatic versioning. Every build, every version, its own URL.
- Toolchain-agnostic. If it runs in Docker and produces files, it works.
- Scales by changing a number. No infrastructure ceremony.
- Self-updating routing. Subdomains appear and disappear as you publish.
- Git-native. Docs and code live together, version together, ship together.

You write the docs. Everything else is handled.
