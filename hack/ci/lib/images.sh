#!/bin/bash

function resolve-image {
  (
    set -euEo pipefail
    shopt -s inherit_errexit

    if [[ $# -ne 1 ]] ; then
      echo 'resolve-image requires exactly 1 argument supplying the image reference'
      exit 1
    fi

    local image_ref
    image_ref="${1}"

    local digest
    digest=$( skopeo inspect --raw docker://"${image_ref}" | skopeo manifest-digest /dev/stdin )
    echo "${image_ref}@${digest}"
  )
}

function resolve-image-simple {
  (
    set -euEo pipefail
    shopt -s inherit_errexit

    if [[ $# -ne 1 ]] ; then
      echo 'resolve-image requires exactly 1 argument supplying the image reference'
      exit 1
    fi

    local image_ref
    image_ref="${1}"

    local digest
    digest=$( skopeo inspect --raw docker://"${image_ref}" | skopeo manifest-digest /dev/stdin )
    echo "${image_ref%:*}@${digest}"
  )
}

function verify-no-tag-exists {
  (
    set -euEo pipefail
    shopt -s inherit_errexit

    if [[ $# -ne 2 ]] ; then
      echo 'verify-no-tag-exists requires exactly 2 arguments: repository reference and the tag'
      exit 1
    fi

    local repo
    repo="${1}"
    local tag
    tag="${2}"

    local tags_list
    tags_list="$( skopeo list-tags "docker://${repo}" )"
    local has_tag
    has_tag="$( jq --arg 't' "${tag}" '.Tags | any(. == $t)' <<< "${tags_list}" )"
    if [[ "${has_tag}" == "false" ]]; then
      exit 0
    else
      exit 1
    fi
  )
}
