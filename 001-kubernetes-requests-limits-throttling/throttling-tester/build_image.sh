#!/bin/bash
set -x
image_name=throttling-example
image_tag=latest
registry_url=ghcr.io/kacerr

docker build -t $image_name:`git rev-parse HEAD` . --build-arg GIT_COMMIT_HASH=`git rev-parse HEAD`

final_tag="$registry_url/$image_name:$image_tag"

docker tag $image_name:`git rev-parse HEAD` $final_tag
docker push $final_tag
echo "Container has been pushed as: ${final_tag}"
