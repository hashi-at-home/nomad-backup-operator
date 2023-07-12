[![pre-commit.ci status](https://results.pre-commit.ci/badge/github/hashi-at-home/nomad-backup-operator/main.svg)](https://results.pre-commit.ci/latest/github/hashi-at-home/nomad-backup-operator/main) [![Release](https://github.com/hashi-at-home/nomad-backup-operator/actions/workflows/release.yml/badge.svg)](https://github.com/hashi-at-home/nomad-backup-operator/actions/workflows/release.yml)

# Nomad backup operator

A small go program using the [Nomad Streaming API](https://www.nomadproject.io/api-docs/events) to create backup jobs for new jobs that are deployed to the cluster.

This is more or less a copy of work previously done by [Andy Davies](https://github.com/Pondidum).
