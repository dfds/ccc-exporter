FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

COPY internal /app/internal
COPY . /app/.

RUN go build -tags=viper_bind_struct -o /app/client /app/cmd/main.go

FROM golang:1.21-alpine

COPY --from=build /app/client /app/client

CMD [ "/app/client" ]