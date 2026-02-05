variable "TAG" {
  default = "latest"
}

group "default" {
  targets = ["builder-mkdocs"]
}

target "builder-mkdocs" {
  context = "./mkdocs-material"
  tags    = ["romain325/doc-thor-builder-mkdocs:${TAG}"]
  annotations = [
    "oci.opencontainers.image.title=doc-thor MkDocs Material builder",
    "oci.opencontainers.image.url=https://github.com/romain325/doc-thor/tree/main/images/mkdocs-material",
    "oci.opencontainers.image.documentation=https://github.com/romain325/doc-thor/tree/main/images/mkdocs-material/README.md",
    "oci.opencontainers.image.description=Builder to generate mkdocs material based static website that can be handled by doc-thor",
    "oci.opencontainers.image.authors=kelkchoz",
    "oci.opencontainers.image.version=${TAG}"
  ]
}
