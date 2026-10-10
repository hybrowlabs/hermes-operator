#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
fail=0
check() { if ! eval "$2"; then echo "FAIL: $1"; fail=1; else echo "ok: $1"; fi; }
check "Dockerfile builder base pinned by digest" "grep -Eq '^FROM .*golang:[0-9.]+@sha256:[0-9a-f]{64} AS builder' Dockerfile"
check "Dockerfile runtime base pinned by digest" "grep -Eq '^FROM gcr.io/distroless/static:nonroot@sha256:[0-9a-f]{64}' Dockerfile"
check "Jenkinsfile exists" "test -f Jenkinsfile"
check "Jenkinsfile defaults to no push" "grep -Eq \"booleanParam\\(name: 'DO_PUSH', defaultValue: false\" Jenkinsfile"
check "Jenkinsfile restricts IMAGE to registry.hybrowlabs.com" "grep -q 'registry\\\\.hybrowlabs\\\\.com' Jenkinsfile"
check "Jenkinsfile rejects latest tag" "grep -q \"IMAGE_TAG == 'latest'\" Jenkinsfile"
check "Jenkinsfile runs go tests before build" "awk '/stage\\(.Unit tests/{a=NR} /stage\\(.Build/{b=NR} END{exit !(a&&b&&a<b)}' Jenkinsfile"
check "Jenkinsfile push credential only inside Push stage" '[ "$(grep -c "credentialsId: .registry-push-hermes-operator." Jenkinsfile)" = 1 ]'
check "Jenkinsfile verifies registry digest after push" "grep -q 'imagetools inspect' Jenkinsfile"
check "Jenkinsfile records SBOM and scan" "grep -q 'syft' Jenkinsfile && grep -q 'trivy image' Jenkinsfile"
exit $fail
