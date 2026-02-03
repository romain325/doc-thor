# Technical and Commercial Documentation: Keep Them Apart

## Nothing derails a developer faster than this scenario

You need to find out how to authenticate with the API. You search. You find a document.
You open it. The first section is about pricing tiers. The second section is a testimonial
from a customer. Somewhere, on page four, after a "Why choose us?" heading, there is a
code example. Maybe.

You have just wasted ten minutes. And you haven't started coding yet.

---

## They are written by different people

Commercial documentation — marketing copy, product pages, feature comparisons for
prospective customers — is written by people who think about conversion, brand voice,
and making the product sound good. That's their job. They do it well.

Technical documentation — API references, integration guides, architecture docs,
runbooks, troubleshooting guides — is written by people who think about precision,
completeness, and "can a developer actually use this to build something." Also their job.
Also done well, ideally.

These are different jobs. Mixing the output of both into the same document serves
neither audience. It just confuses both.

---

## They change at completely different rates

Marketing copy changes when the product strategy changes. Quarterly, maybe. Less, if
you're lucky. It moves at the speed of business decisions.

Technical documentation changes when the code changes. Which is: constantly.
A developer pushes a change on Tuesday. The API behaves differently on Tuesday.
The documentation should reflect that on Tuesday. Not next sprint. Tuesday.

If both types of documentation live in the same system, with the same update cadence,
one of them is always out of date. The answer is always the technical docs.
Because marketing has a dedicated writer. Engineering has "someone will update it
when they get a chance." They never get a chance.

---

## The audience problem

A potential customer reading your documentation wants to know: "Can this product do
what I need? How much does it cost? What does onboarding look like? Who else uses it?"

A developer integrating your API wants to know: "What are the exact parameters for
this endpoint? What does error code 4032 mean? What is the rate limit? What changed
between v2.1 and v2.2?"

These are not the same questions. They don't belong on the same page. They don't benefit
from the same structure, the same tone, or the same update process. Putting them together
doesn't create a "comprehensive documentation experience." It creates a confusing one
where everyone finds less of what they need.

---

## The practical answer

- **Commercial documentation** lives on your marketing site. Managed by your marketing
  team. Updated on their schedule. Optimized for prospective users and search engines.

- **Technical documentation** lives in (or very near) your code repositories. Built and
  versioned with your releases. Updated when the code changes. Served at a dedicated
  technical documentation URL with no marketing copy in sight.

doc-thor handles the second category. The first category is a different problem entirely —
and it should stay that way.
