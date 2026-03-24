# romain325/doc-thor-builder-vuepress

Official doc-thor image for [VuePress](https://vuepress.vuejs.org/) projects.

## Container contract

All doc-thor build images follow the same interface with the builder:

| Mount     | Mode       | Purpose                                            |
| --------- | ---------- | -------------------------------------------------- |
| `/repo`   | read-only  | Cloned source repository                           |
| `/output` | read-write | Build output — collected by the builder after exit |

Exit code `0` signals success. Anything else is a failure.

## Test locally

```sh
docker run --rm \
  -v $(pwd)/path/to/docs:/repo:ro \
  -v /tmp/out:/output \
  romain325/doc-thor-builder-vuepress
```

Output will appear in `/tmp/out`.

## Extending

Need extra plugins? Create a new image that extends this one:

```dockerfile
FROM romain325/doc-thor-builder-vuepress

```
