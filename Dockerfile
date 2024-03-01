FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

COPY internal /app/internal
COPY . /app/.
COPY conf /app/conf

RUN go build -o /app/client /app/cmd/main.go

FROM golang:1.21-alpine

COPY --from=build /app/client /app/client

CMD [ "/app/client" ]