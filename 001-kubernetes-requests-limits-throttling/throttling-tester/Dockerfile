# builder image
FROM golang:1.16.5-buster as builder
RUN mkdir /build
ADD / /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -a -o throttling-tester .


# generate clean, final image for end users
FROM debian:buster
COPY --from=builder /build/throttling-tester .

# executable
ENTRYPOINT [ "./throttling-tester" ]
# arguments that can be overridden
# CMD [ "3", "300" ]