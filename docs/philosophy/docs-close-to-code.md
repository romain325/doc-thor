# Keep Your Docs Next to Your Code

## Your documentation is already lying to you

Open your project's installation process.
Now try to actually install it that way.

If it worked on the first try, congratulations. You are in the rare minority of projects
where someone actually maintained the docs. For the rest of you: the docs lied. Not on
purpose. They just weren't updated when the code changed, because they lived somewhere
else, and nobody thought to go update them.

This is not a bug. It is the predictable outcome of putting documentation far from the
thing it describes.

---

## The staleness problem

Documentation rots. This isn't a moral failing — it's physics. Code changes. Documentation,
if it lives somewhere other than right next to the code, does not change at the same rate.
The gap grows. Eventually, the documentation describes a system that no longer exists.

The reason is simple: when you change code, your editor is open, your context is loaded,
you are thinking about that specific piece of the system. If the docs are in the same
directory, in the same PR, they are right there. If they are in a separate wiki, in a
separate tool, with a separate login — they are not there. And you won't go there. Not
right now. Maybe later.

Later never comes. It never does.

---

## Docs next to code means docs in the PR

This is the real win. When documentation lives in the same repository as the code it
describes, it is part of the same pull request. Someone changes an API? The doc update
goes in the same PR. Someone reviews the PR? They see both the code change and the doc
change. If the docs are wrong or missing, the review catches it.

This is not a workflow hack. It is the only workflow that actually works at scale.
Everything else is just hoping someone remembers to update the wiki.

---

## "But our docs need more structure than a README"

True. So use a `docs/` folder in your repository. Put an MkDocs config in there.
Build it into a proper site with navigation, search, and cross-references.
You get the structure of a documentation website and the version control of being
in the repo. You don't have to choose one or the other.

That's exactly what doc-thor is built around.

---

## The rule

If documentation describes code, it lives with the code. No exceptions.
No "but it's easier to edit in the wiki." Easier to edit is not the same as easier
to keep correct. Those are different things, and conflating them is how documentation
dies.
