#
## build stage
FROM golang:1.20 as builder

LABEL maintainer="github.com/SlevinWasAlreadyTaken"

WORKDIR /app

# modules
ARG CGO_ENABLED=0
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# copy sources
COPY . .

# compile
RUN go build -o /bin/gha-file-sync

#
## run stage - minimalist final image
FROM alpine:3.17

# ensure glibc binaries is runnable by install glibc compatibility packages
RUN apk add libc6-compat

# get binary from build image
COPY --from=builder /bin/gha-file-sync /bin/gha-file-sync

ENTRYPOINT ["/bin/gha-file-sync"]
