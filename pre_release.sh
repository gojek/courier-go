#!/bin/bash

set -e

declare COURIER_TAG

help() {
   printf "\n"
   printf "Usage: %s [-t tag]\n" "$0"
   printf "\t-t New Courier unreleased tag. Updates all submodule go.mod files with this tag.\n"
   exit 1 # Exit script after printing help
}

while getopts "t:" opt
do
   case "$opt" in
      t ) COURIER_TAG="$OPTARG" ;;
      ? ) help ;; # Print help
   esac
done

declare -r SEMVER_REGEX="^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(\\-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

validate_tag() {
    local tag_=$1
    if [[ "${tag_}" =~ ${SEMVER_REGEX} ]]; then
	    printf "%s is valid semver tag.\n" "${tag_}"
    else
	    printf "%s is not a valid semver tag.\n" "${tag_}"
	    return 1
    fi
}

# Print help in case parameter is empty
if [[ -z "$COURIER_TAG" ]]
then
    printf "parameter '-t' must be specified.\n"
    help
fi


## Validate tags first
validate_tag "${COURIER_TAG}" || exit $?
TAG_FOUND=$(git tag --list "${COURIER_TAG}")
if [[ ${TAG_FOUND} = "${COURIER_TAG}" ]] ; then
    printf "Tag %s already exists in this repo\n" "${COURIER_TAG}"
    exit 1
fi

# Get version for version.go
COURIER_VERSION=$(echo "${COURIER_TAG}" | egrep -o "${SEMVER_REGEX}")
# Strip leading v
COURIER_VERSION="${COURIER_VERSION#v}"

cd "$(dirname "$0")"

if ! git diff --quiet; then \
    printf "Working tree is not clean, can't proceed\n"
    git status
    git diff
    exit 1
fi

# Update version.go version definition
cp version.go version.go.bak
sed "s/\(return \"\)[0-9]*\.[0-9]*\.[0-9]*\"/\1${COURIER_VERSION}\"/" ./version.go.bak > ./version.go
rm -f ./version.go.bak

declare -r BRANCH_NAME=pre_release_${COURIER_TAG}

patch_gomods() {
    local pkg_=$1
    local tag_=$2
    # now do the same for all the directories underneath
    PACKAGE_DIRS=$(find . -mindepth 2 -type f -name 'go.mod' -exec dirname {} \; | egrep -v 'tools' | sed 's|^\.\/||' | sort)
    # quote any '.' characters in the pkg name
    local quoted_pkg_=${pkg_//./\\.}
    for dir in $PACKAGE_DIRS; do
	    cp "${dir}/go.mod" "${dir}/go.mod.bak"
	    sed "s|${quoted_pkg_}\([^ ]*\) v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*[^0-9]*.*$|${pkg_}\1 ${tag_}|" "${dir}/go.mod.bak" >"${dir}/go.mod"
	    rm -f "${dir}/go.mod.bak"
    done
}

# branch off from existing master
git checkout -b "${BRANCH_NAME}" master

if [ -n "${COURIER_TAG}" ]; then
    patch_gomods ***REMOVED*** "${COURIER_TAG}"
fi

# Run lint to update go.sum
make lint

# Add changes and commit.
git add .
make ci

declare COMMIT_MSG=""
COMMIT_MSG+="Releasing ${COURIER_TAG}"
git commit -m "${COMMIT_MSG}"

printf "Now run following to verify the changes.\ngit diff master\n"
printf "\nThen push the changes to upstream\n"
