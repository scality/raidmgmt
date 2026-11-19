FROM mcr.microsoft.com/devcontainers/go:1.25-trixie@sha256:d2f4cf226b60c97800b485ef9e226f50e211c9c83414cbd13b6aa27c7fd994ca

USER vscode

# Same version as the CI lint job.
# renovate: datasource=github-releases depName=golangci/golangci-lint extractVersion=^v(?<version>.+)$
ARG GOLANGCI_VERSION=2.15.0
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
  | sudo sh -s -- v${GOLANGCI_VERSION}
