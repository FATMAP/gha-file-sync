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

FROM scratch

# get binary from build image
COPY --from=builder /bin/git-file-sync /bin/git-file-sync

ENTRYPOINT ["/bin/git-file-sync"]
