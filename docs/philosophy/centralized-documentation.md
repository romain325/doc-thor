# The Case for Centralized Documentation

## Your documentation is scattered. Admit it

Some of it is in a Git repo. Some of it is in a CMS. Some of it is in a Notion page
that three people bookmarked and one person actually reads. One critical piece — the one
you will need at 2 AM on a Friday — is in an email thread from 2021.

Congratulations. You have documentation. It is just not findable.

---

## Why scatter is a trap

When documentation lives in multiple places, it doesn't just become hard to find.
It becomes hard to **trust**. Which version is current? Was the CMS page updated
after the last refactor? Does the README reflect actual behavior, or the behavior from
six months ago?

The answer, statistically, is: the one you are looking at is wrong.

Scattered documentation doesn't fail loudly. It fails quietly. A developer reads stale docs,
implements something based on outdated assumptions, and spends two days debugging a problem
that a single source of truth would have prevented in two minutes. Nobody writes a postmortem
for that. It just becomes "the thing we always have to figure out ourselves."

---

## What centralization actually means

It does not mean "dump everything into one giant wiki and pray someone organizes it."
It means having **one authoritative location** for each piece of documentation,
with a clear path to find it.

For technical documentation, that location is close to the code. For project overviews,
it might be a dedicated docs site. The point is: you know where to look, and what you find
there is correct. Revolutionary concept, apparently.

---

## The cost of not centralizing

- **Onboarding time** goes up. New developers spend days hunting for docs that should take
  minutes to find. They eventually learn to ask people instead of reading docs. Then the
  people become a single point of failure.

- **Maintenance burden** multiplies. Every copy of a document is a document that can become
  stale independently. You don't have one wrong doc. You have four, and you don't know
  which one is the least wrong.

- **Trust erodes.** Once developers learn that docs might be wrong, they stop reading them.
  Then you have no documentation at all — just code with comments that say
  `// TODO: figure out why this works`.

---

## What doc-thor does about it

It doesn't solve the "write good documentation" problem. Nothing automates that.
What it does is make sure that once documentation exists, it gets built, versioned,
published, and served — consistently, reliably, every time.
The infrastructure stops being a reason to skip documenting things.
