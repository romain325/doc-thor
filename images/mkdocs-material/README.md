# doc-thor/builder-mkdocs

Official doc-thor image for [MkDocs](https://www.mkdocs.org/) projects. Includes the [Material theme](https://squidfunk.github.io/mkdocs-material/).

## Container contract

All doc-thor build images follow the same interface with the builder:

| Mount     | Mode       | Purpose                                            |
| --------- | ---------- | -------------------------------------------------- |
| `/repo`   | read-only  | Cloned source repository (contains `mkdocs.yml`)   |
| `/output` | read-write | Build output — collected by the builder after exit |

Exit code `0` signals success. Anything else is a failure.

## Build

```sh
docker build -t doc-thor/builder-mkdocs .
```

## Test locally

```sh
docker run --rm \
  -v $(pwd)/path/to/docs:/repo:ro \
  -v /tmp/out:/output \
  doc-thor/builder-mkdocs
```

Output will appear in `/tmp/out`.

## Extending

Need extra plugins? Create a new image that extends this one:

```dockerfile
FROM doc-thor/builder-mkdocs

RUN pip install --no-cache-dir mkdocs-git-revision-date-plugin
```
