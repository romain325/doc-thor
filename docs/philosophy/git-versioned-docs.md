# Git Is Already Doing the Hard Part

## Your code has a time machine. Your docs don't

Every line of code in your repository has a history. You can see who wrote it, when,
why, and what it looked like before. You can roll back to any point in time. You can
branch, compare, bisect. You have an entire forensic toolkit built into your daily
workflow.

Your documentation — if it lives in a wiki or a CMS — has none of that. Maybe it has
"last edited by" and a timestamp. Maybe it has a revision history that looks like
"v1, v1.1, v1.1-final, v1.1-final-FINAL, v1.1-final-FINAL-2, v2."
That is not version control. That is organized panic.

---

## What git gives you for free (if you put docs in the repo)

**Every version of your docs matches a version of your code.** When you tag a release,
the documentation for that release is tagged too. Not approximately. Not "close enough."
Exactly. The docs and the code are the same commit.

**You can see what changed and why.** A commit message that says
"update auth docs for new token format" is infinitely more useful than a wiki edit
log that says "updated." You get the context. You get the diff. You get the author.

**Branches work for docs too.** Writing documentation for an upcoming feature?
Do it on the feature branch. It gets merged when the feature gets merged.
No coordination meeting. No "oh wait, did someone update the docs for that?"

**You can blame a documentation bug** the same way you blame a code bug.
`git blame` works on Markdown files. It works beautifully, actually.

---

## The "but git is hard" argument

It is not, anymore. If your developers use git daily — and they do — then contributing
to documentation in the same repository requires exactly zero additional tooling.
The barrier to a documentation contribution is identical to the barrier to a code
contribution: open a branch, make a change, submit a PR.

Compare that to "navigate to the wiki, find the right page, click edit, hope someone
notices if it's wrong." Git wins. It is not even close.

---

## Versioned docs for versioned software

If you ship versions of your software, you need versions of your documentation.
Not just "the latest docs" — the docs that match what a user actually has installed.
A developer running v2.1 should not be reading documentation that was written for v3.0-beta.

When docs live in the repository and you build them per-version, this is automatic.
Tag a release. Build the docs for that tag. Publish them. Done.
No manual "freeze the documentation for this version" process.
No doc-version mismatch bugs that nobody notices until someone files a support ticket
at 11 PM.

doc-thor does exactly this: builds docs per version, publishes per version,
serves any version at any time. Git does the hard part. doc-thor does the rest.
