#! /bin/bash -x

PROJECT_VERSION=$1
PROJECT_NAME=$2
BUILD_NAME=${PROJECT_NAME:-"techstack"}
PACKAGE_NAME="github.com/alayou/techstack"
Now="$(date '+%Y%m%d_%I%M%S')"
BuiltAt="$(date '+%Y-%m-%dT%I:%M:%S')"
Revision="$(git rev-parse --short HEAD)"
Tag="$(git describe --tags)"

echo built $(pwd)
echo BUILD_NAME="$BUILD_NAME" Now="$Now" Tag="$Tag" Revision="$Revision"
go mod tidy

export GOOS=windows
export GOARCH=amd64
go build -ldflags "-X main.version=${PROJECT_VERSION:-"$Now"} -X main.revision=${Revision} -X main.builtAt=${BuiltAt}" -o "build/${BUILD_NAME}.exe" "${PACKAGE_NAME}"

export GOOS=linux
export GOARCH=amd64
go build -ldflags "-X main.version=${PROJECT_VERSION:-"$Now"} -X main.revision=${Revision} -X main.builtAt=${BuiltAt}" -o "build/${BUILD_NAME}-linux" "${PACKAGE_NAME}"
go build -ldflags "-X main.version=${PROJECT_VERSION:-"$Now"} -X main.revision=${Revision} -X main.builtAt=${BuiltAt}" -o "build/${BUILD_NAME}" "${PACKAGE_NAME}"

export GOOS=linux
export GOARCH=arm64
go build -ldflags "-X main.version=${PROJECT_VERSION:-"$Now"} -X main.revision=${Revision} -X main.builtAt=${BuiltAt}" -o "build/${BUILD_NAME}-linux-arm64" "${PACKAGE_NAME}"
