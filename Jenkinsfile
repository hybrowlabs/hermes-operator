// HYBA-2345: hybrowlabs/hermes-operator fork build on the rootless docker-build agent.
// Default is a no-push build: unit tests, image build, SBOM, scan. Push needs DO_PUSH=true, an approved R1 line and credential registry-push-hermes-operator.
pipeline {
  agent { label 'docker-build' }
  parameters {
    string(name: 'GIT_SHA', defaultValue: '', description: 'Commit (40 hex) of hybrowlabs/hermes-operator to build')
    string(name: 'IMAGE', defaultValue: 'registry.hybrowlabs.com/hermes/hermes-operator', description: 'Registry/repo, no tag')
    string(name: 'IMAGE_TAG', defaultValue: '', description: 'Plain tag, for example v0.2.0-secretns1. Never latest')
    booleanParam(name: 'DO_PUSH', defaultValue: false, description: 'Push after a green build. Needs an approved R1 login in credential registry-push-hermes-operator')
  }
  options { timestamps(); timeout(time: 60, unit: 'MINUTES'); buildDiscarder(logRotator(numToKeepStr: '30')) }
  environment {
    CI_WORKSPACE = "${env.WORKSPACE}"
    CI_IMAGES = "${params.IMAGE}:${params.IMAGE_TAG}"
    CI_REGISTRY = "${params.IMAGE.tokenize('/')[0]}"
    CI_S3_ENDPOINT = 'https://s3.ap-southeast-1.wasabisys.com'
    CI_S3_REGION = 'ap-southeast-1'
    CI_BUCKET = 'hybrow-ci-artifacts'
    CI_KEY_PREFIX = "hermes-operator/${params.GIT_SHA}/${env.BUILD_NUMBER}"
  }
  stages {
    stage('Disk guard') {
      steps {
        sh '''set -eu
          . /opt/ci-build/bin/ci-env
          ci-disk-guard
          ci-verify
        '''
      }
    }
    stage('Preflight') {
      steps {
        script {
          if (!(params.GIT_SHA ==~ /[0-9a-f]{40}/)) { error('GIT_SHA must be 40 lowercase hex characters') }
          if (!(params.IMAGE ==~ /registry\.hybrowlabs\.com\/[a-z0-9._\/-]+/)) { error('IMAGE must be on registry.hybrowlabs.com') }
          if (!(params.IMAGE_TAG ==~ /[A-Za-z0-9_.-]{1,64}/) || params.IMAGE_TAG == 'latest' || params.IMAGE_TAG == 'stable') { error('IMAGE_TAG must be a plain tag and not latest or stable') }
        }
      }
    }
    stage('Clean workspace') {
      steps { deleteDir() }
    }
    stage('Checkout') {
      steps {
        checkout([$class: 'GitSCM', branches: [[name: params.GIT_SHA]], userRemoteConfigs: [[url: 'https://github.com/hybrowlabs/hermes-operator.git']]])
        sh '''#!/usr/bin/env bash
          set -euo pipefail
          . /opt/ci-build/bin/ci-env
          head="$(git rev-parse HEAD)"
          [ "$head" = "$GIT_SHA" ] || { echo "HEAD $head is not GIT_SHA $GIT_SHA"; exit 1; }
          echo "commit: $head"
          DOCKER_CONFIG="$(ci-docker-config create)"
          echo "$DOCKER_CONFIG" > .docker-config-path
        '''
      }
    }
    stage('CI self-test') {
      steps {
        sh '''#!/usr/bin/env bash
          set -euo pipefail
          bash ci/tests/test-ci.sh
        '''
      }
    }
    stage('Unit tests') {
      steps {
        sh '''#!/usr/bin/env bash
          set -euo pipefail
          . /opt/ci-build/bin/ci-env
          DOCKER_CONFIG="$(cat .docker-config-path)"; export DOCKER_CONFIG
          docker run --rm -v "$PWD":/src -w /src golang:1.27@sha256:e432b43af23a9328d56a7c499be0476810aa344acbcf65fc7c455d4ff5a40602 \
            sh -c 'go vet ./cmd ./internal/... && go test ./internal/cachescope ./internal/webhook ./internal/oci ./internal/resources ./api/... && go test ./internal/controller -skip TestControllers' > unit-tests.log 2>&1 || { cat unit-tests.log; exit 1; }
          cat unit-tests.log
        '''
      }
    }
    stage('Build') {
      steps {
        sh '''#!/usr/bin/env bash
          set -euo pipefail
          . /opt/ci-build/bin/ci-env
          DOCKER_CONFIG="$(cat .docker-config-path)"; export DOCKER_CONFIG
          created="$(date -u +%FT%TZ)"
          docker buildx build --progress=plain --load --provenance=false \
            --label "org.opencontainers.image.revision=$GIT_SHA" --label "org.opencontainers.image.created=$created" \
            -t "$IMAGE:$IMAGE_TAG" . > build.log 2>&1 || { cat build.log; exit 1; }
          cat build.log
        '''
      }
    }
    stage('SBOM and scan') {
      steps {
        sh '''set -eu
          . /opt/ci-build/bin/ci-env
          DOCKER_CONFIG="$(cat .docker-config-path)"; export DOCKER_CONFIG
          syft "$IMAGE:$IMAGE_TAG" -o spdx-json=sbom.spdx.json
          trivy image --timeout 30m --format json --output trivy-report.json --exit-code 0 "$IMAGE:$IMAGE_TAG"
          trivy image --timeout 30m --severity CRITICAL --exit-code 0 --no-progress "$IMAGE:$IMAGE_TAG" > trivy-critical.txt 2>&1 || true
          cat trivy-critical.txt
        '''
      }
    }
    stage('Push') {
      when { expression { params.DO_PUSH } }
      steps {
        withCredentials([usernamePassword(credentialsId: 'registry-push-hermes-operator', usernameVariable: 'REG_USER', passwordVariable: 'REG_PASS')]) {
          sh '''#!/usr/bin/env bash
            set -euo pipefail
            . /opt/ci-build/bin/ci-env
            DOCKER_CONFIG="$(cat .docker-config-path)"; export DOCKER_CONFIG
            printf '%s' "$REG_PASS" | docker login "$CI_REGISTRY" -u "$REG_USER" --password-stdin
            docker push "$IMAGE:$IMAGE_TAG" > push.log 2>&1 || { cat push.log; docker logout "$CI_REGISTRY" || true; exit 1; }
            cat push.log
            docker image inspect --format '{{index .RepoDigests 0}}' "$IMAGE:$IMAGE_TAG" > digest.txt
            reg_digest="$(docker buildx imagetools inspect "$IMAGE:$IMAGE_TAG" --format '{{.Manifest.Digest}}')"
            [ "$(cut -d@ -f2 digest.txt)" = "$reg_digest" ] || { echo "registry digest differs from push digest"; exit 1; }
            echo "registry read-back digest: $reg_digest"
            docker logout "$CI_REGISTRY"
          '''
        }
      }
    }
  }
  post {
    always {
      catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
        withEnv(["BUILD_RESULT=${currentBuild.currentResult}"]) {
          withCredentials([usernamePassword(credentialsId: 'ci-artifacts-s3', usernameVariable: 'CI_S3_ACCESS_KEY_ID', passwordVariable: 'CI_S3_SECRET_ACCESS_KEY')]) {
            sh '''set -eu
              . /opt/ci-build/bin/ci-env
              digest="$(cat digest.txt 2>/dev/null || echo none)"
              files=""
              for f in unit-tests.log build.log push.log sbom.spdx.json trivy-report.json trivy-critical.txt; do
                if [ -f "$f" ]; then files="$files $f"; fi
              done
              # shellcheck disable=SC2086
              ci-upload $files
            '''
          }
        }
      }
      sh '''. /opt/ci-build/bin/ci-env
        DOCKER_CONFIG="$(cat .docker-config-path 2>/dev/null || true)"; export DOCKER_CONFIG
        /opt/ci-build/bin/ci-cleanup'''
    }
  }
}
