#!/bin/bash

function tag_from_gh_ref {
    local tmp_file
    tmp_file=$( mktemp )
    trap 'rm -f "${tmp_file}"' RETURN
    echo "${1}" > "${tmp_file}"
    (
        sed -i -E -e '/^refs\/heads\/master$/,${s//latest/;b};$q1' "${tmp_file}" || \
        sed -i -E -e '/^refs\/heads\/release-([0-9]+\.[0-9]+)$/,${s//\1/;b};$q1' "${tmp_file}" || \
        sed -i -E -e '/^refs\/tags\/v([0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)\.[0-9]+)?)$/,${s//\1/;b};$q1' "${tmp_file}" || \
        false
    ) && cat "${tmp_file}"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euEo pipefail
  shopt -s inherit_errexit

  message_prefix="Testing parsing tags from GitHub refs"

  echo "${message_prefix}..." > /dev/stderr
  function err_handler {
    echo "${message_prefix}...FAILED! Error on line ${1}. Use \`bash -x ${0}\` to see more details." > /dev/stderr
  }
  trap 'err_handler "${LINENO}"' ERR

  [[ "$( tag_from_gh_ref 'refs/heads/master' )" == 'latest' ]]
  [[ "$( tag_from_gh_ref 'refs/heads/release-1.0' )" == '1.0' ]]
  [[ "$( tag_from_gh_ref 'refs/tags/v1.0.0-alpha.0' )" == '1.0.0-alpha.0' ]]
  [[ "$( tag_from_gh_ref 'refs/tags/v1.0.0-beta.0' )" == '1.0.0-beta.0' ]]
  [[ "$( tag_from_gh_ref 'refs/tags/v1.0.0-rc.0' )" == '1.0.0-rc.0' ]]
  [[ "$( tag_from_gh_ref 'refs/tags/v1.0.0' )" == '1.0.0' ]]
  ( (tag_from_gh_ref '' && exit 1) || true )
  ( (tag_from_gh_ref 'foo' && exit 1) || true )
  ( (tag_from_gh_ref 'master' && exit 1) || true )
  ( (tag_from_gh_ref '/refs/heads/master' && exit 1) || true )
  ( (tag_from_gh_ref 'refs/heads/masters' && exit 1) || true )
  ( (tag_from_gh_ref 'refs/heads/1.0' && exit 1) || true )
  ( (tag_from_gh_ref 'refs/heads/v1.0' && exit 1) || true )
  ( (tag_from_gh_ref 'v1.0.0' && exit 1) || true )
  ( (tag_from_gh_ref 'v1.0.0-rc.0' && exit 1) || true )
  ( (tag_from_gh_ref 'refs/tags/v1.0.0-alpha' && exit 1) || true )
  ( (tag_from_gh_ref 'refs/tags/v1.0.0-alpha.' && exit 1) || true )
  ( (tag_from_gh_ref 'refs/tags/v1.0.0-alpha0' && exit 1) || true )
  ( (tag_from_gh_ref 'refs/tags/v1.0.0.0' && exit 1) || true )
  ( (tag_from_gh_ref 'refs/heads/release-1.0.0.0' && exit 1) || true )

  echo "${message_prefix}..SUCCESS." >&2
fi
