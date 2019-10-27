FROM golang:1.13.3-buster as builder

RUN apt-get update && apt-get -y install libopus-dev libopusfile-dev

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY . .

RUN go build -o main .
FROM golang:1.13.3-buster as runner

ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

RUN apt-get update && apt-get -y install  python-pip libopus0 libopusfile0 ffmpeg
RUN pip install --upgrade pip
RUN pip install youtube_dl

WORKDIR /app
COPY --from=builder /app/main ./

RUN chmod +x ./main
CMD ["/app/main"]

