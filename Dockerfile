# Static resources compilation stage
FROM node:22.16-alpine3.22 AS build-static
# Create working directory, this is the path where the application code is stored inside the container
WORKDIR /home/go-flarum
COPY package.json *.lock ./
# Only install production dependencies
# The node image comes with yarn pre-installed
# ## BOF CLEAN
# # Chinese users may need to set a mirror registry
# ARG registry=https://registry.npmmirror.com/
# ARG disturl=https://npm.taobao.org/dist
# RUN yarn config set disturl $disturl
# RUN yarn config set registry https://registry.yarnpkg.com
# ## EOF CLEAN
RUN yarn --only=prod
COPY webpack.config.js ./
COPY view ./view
RUN yarn build

# Golang compilation stage
FROM golang:1.23.10-alpine3.22 AS build-backend
# All these steps will be cached
WORKDIR /home/go-flarum
# ## BOF CLEAN
# # Chinese users may need to set up a Go proxy and faster Alpine mirrors
#RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
#RUN go env -w GOPROXY=https://goproxy.cn,direct
# ## EOF CLEAN
RUN apk update && apk add git
# COPY go.mod and go.sum files to the workspace
COPY go.mod .
COPY go.sum .
RUN go mod download
# COPY the source code as the last step
COPY . .
# Build the binary
RUN GIT_COMMIT=$(git rev-list -1 HEAD) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags "-X main.GitCommit=$GIT_COMMIT" \
    -o go-flarum ./cmd/server/main.go
RUN GIT_COMMIT=$(git rev-list -1 HEAD) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags "-X main.GitCommit=$GIT_COMMIT" \
    -o go-flarum-migration ./cmd/migration/main.go

# Build the final image
FROM alpine:3.22
WORKDIR /home/go-flarum
COPY ./view view
COPY ./config/config.yaml-docker config.yml
COPY --from=build-static /home/go-flarum/static webpack/static
COPY --from=build-backend /home/go-flarum/go-flarum /usr/local/bin/go-flarum
COPY --from=build-backend /home/go-flarum/go-flarum-migration /usr/local/bin/go-flarum-migration
EXPOSE 8082
CMD ["/usr/local/bin/go-flarum", "-config", "/home/go-flarum/config.yml"]