#!/bin/bash

# Parameter
OWNER="hezhijie0327"
REPO="cloudflareddns"
TAG="latest"

## Function
# Get Latest Image
function GetLatestImage() {
    docker pull ${OWNER}/${REPO}:${TAG} && IMAGES=$(docker images -f "dangling=true" -q)
}
# Cleanup Current Container
function CleanupCurrentContainer() {
    if [ $(docker ps -a --format "table {{.Names}}" | grep -E "^${REPO}$") ]; then
        docker stop ${REPO} && docker rm ${REPO}
    fi
}
# Create New Container
function CreateNewContainer() {
    docker run --name ${REPO} --net host --restart=always \
        -v /etc/resolv.conf:/etc/resolv.conf:ro \
        -e XAUTHEMAIL="demo@zhijie.online" \
        -e XAUTHKEY="123defghijk4567pqrstuvw890" \
        -e ZONENAME="zhijie.online" \
        -e RECORDNAME="demo.zhijie.online" \
        -e TYPE="A_AAAA" \
        -e TTL="1" \
        -e STATICIP="auto" \
        -e PROXYSTATUS="false" \
        -e RUNNINGMODE="update" \
        -e UPDATEFREQUENCY="900" \
        -d ${OWNER}/${REPO}:${TAG}
}
# Cleanup Expired Image
function CleanupExpiredImage() {
    if [ "${IMAGES}" != "" ]; then
        docker rmi ${IMAGES}
    fi
}

## Process
# Call GetLatestImage
GetLatestImage
# Call CleanupCurrentContainer
CleanupCurrentContainer
# Call CreateNewContainer
CreateNewContainer
# Call CleanupExpiredImage
CleanupExpiredImage
