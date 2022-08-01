FROM golang:1.18-alpine as build
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . ./
RUN go build -o main-auth-server .

FROM alpine:latest
COPY --from=build /app/main-auth-server .

CMD [ "./main-auth-server" ]