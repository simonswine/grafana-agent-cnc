FROM node:20.11.0 as build-nodejs

WORKDIR /app

COPY ./frontend/package.json ./frontend/package-lock.json ./

RUN npm install

COPY ./frontend/tsconfig.json ./frontend/index.html ./
COPY ./frontend/src ./src

RUN npm run build

FROM golang:1.21 AS build-go

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY --from=build-nodejs /app/dist ./frontend/dist
COPY ./frontend/frontend.go ./frontend
COPY ./app ./app
COPY ./model ./model
COPY ./main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o grafana-agent-cnc .

FROM alpine:3.19.1

USER nobody:nobody

COPY --from=build-go /app/grafana-agent-cnc /usr/bin/grafana-agent-cnc

EXPOSE 8333
ENTRYPOINT ["/usr/bin/grafana-agent-cnc"]
