#!/usr/bin/env bash

check_image_tag_exists() {
  repository_name="$1"
  image_tag="$2"

  CMD="$(aws ecr describe-images --repository-name="$repository_name" --image-ids=imageTag="$image_tag" ||:)"

  if [[ ! -z "$CMD" ]]; then
    echo "True"
  else
    echo "False"
  fi

}

abort_process_if_image_tag_exists() {

  repository_name="$1"
  image_tag="$2"

  result=$(check_image_tag_exists "$repository_name" "$image_tag")
  if [[ $result == "True" ]]; then
    echo "Aborting process. Image tag '$image_tag' exists in repository '$repository_name'."
    exit 1
  fi
}

shorten_commit_sha() {
  commit_sha=$1
  echo "${commit_sha}" | cut -c1-7
}
