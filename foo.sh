CIRCLE_TAG="v2.33.444-rc5"
CIRCLE_BRANCH="release/v2.201811/"
CIRCLE_PR_NUMBER="123"
CIRCLE_BUILD_NUM=456

# If the branch being built contains one or more slash `prefix/`,
# use that `prefix` as the tag for the cached CI image.
if [[ $CIRCLE_BRANCH =~ (.*)/[^/]* ]]; then
	VENDOR_CACHE_TAG=${BASH_REMATCH[1]}
else
	VENDOR_CACHE_TAG="master"
fi

# Obtain or build a cached vendor image, which contains the base runtime
# and versioned project dependencies.
echo docker build . \
  --file ./v2/build/Dockerfile \
	--target vendor \
	--tag liveramp/gazette-vendor-cache:${VENDOR_CACHE_TAG} \
  --cache-from liveramp/gazette-vendor-cache:${VENDOR_CACHE_TAG}

# If we're not building for a PR and CACHE_TAG is a prefix of the current
# branch, then push the vendor image for the use of future builds.
if [[ ! -z $CIRCLE_PR_NUMBER && $CIRCLE_BRANCH = ${CACHE_TAG}* ]]; then
  echo docker push liveramp/gazette-vendor-cache:${VENDOR_CACHE_TAG}
fi

echo "build the things"

if [[ $CIRCLE_BRANCH =~ release/v([[:digit:]]+).([[:digit:]]+)/ ]]; then
	RELEASE_TAG="v${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.${CIRCLE_BUILD_NUM}"

  echo docker tag  liveramp/gazette-examples:latest liveramp/gazette-examples:${RELEASE_TAG}
	echo docker push liveramp/gazette-examples:${RELEASE_TAG}

  echo docker tag  liveramp/gazette:latest liveramp/gazette:${RELEASE_TAG}
	echo docker push liveramp/gazette:${RELEASE_TAG}
fi
