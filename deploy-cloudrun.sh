#!/usr/bin/env bash

set -eux

cd "$(dirname "$0")"

set +x

echo "GCP_PROJECT_ID: ${GCP_PROJECT_ID}, GCP_IMAGE_TAG: ${GCP_IMAGE_TAG}, TAG: ${TAG}"

add_common_args()
{
  local -n array=$1
  array+=(
    "--image=${GCP_IMAGE_TAG}" \
    "--tag=${TAG}" \
    "--service-account=cloudrun-runtime@${GCP_PROJECT_ID}.iam.gserviceaccount.com" \
    "--region=us-central1" \
    "--platform=managed" \
    "--set-secrets=ATPROTO_BOT_HANDLE=ATPROTO_BOT_HANDLE:latest" \
    "--set-secrets=ATPROTO_BOT_PASSWORD=ATPROTO_BOT_PASSWORD:latest"
  )
}

deploy_cloud_run()
{
  cloud_run_name=$1
  _args=("$@")
  args=("${_args[@]:1}")

  add_common_args args

  cmd=(
    gcloud \
      "--project=${GCP_PROJECT_ID}" "--quiet" \
      beta run deploy "$cloud_run_name"
  )
  cmd+=( "${args[@]}" )

  # declare -p cmd
  "${cmd[@]}"

  echo "deployed..."
}

update_traffic_cloud_run()
{
  cloud_run_name=$1
  gcloud "--project=${GCP_PROJECT_ID}" "--quiet" run services update-traffic \
    "$cloud_run_name" "--region=us-central1" "--to-tags=${TAG}=100"
}

project_common_args=("--no-traffic" "--max-instances=1")
deploy_cloud_run "server" "--allow-unauthenticated" "--cpu=1" "--memory=512Mi" "--concurrency=40" "--execution-environment=gen2" "--labels=app=vvvot" "${project_common_args[@]}"
update_traffic_cloud_run "server"
