# build stage
FROM golang:1.19 as builder
LABEL maintainer="github.com/SlevinWasAlreadyTaken"

WORKDIR /app

# Modules
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy sources
COPY . .

RUN go build -o /bin/git-file-sync

# run stage - minimalist final image
FROM alpine:3.17

# get binary from build image
COPY --from=builder /bin/git-file-sync /bin/git-file-sync

RUN ls -l /bin

ENTRYPOINT ["/bin/git-file-sync"]
