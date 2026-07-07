FROM node:20-alpine AS frontend-builder
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

FROM golang:1.22-alpine AS backend-builder
WORKDIR /build
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN go build -o /excelper-server ./main.go

FROM alpine:3.20
WORKDIR /app
RUN mkdir -p /data
COPY --from=backend-builder /excelper-server /app/excelper-server
COPY --from=frontend-builder /frontend/dist /app/frontend-dist
EXPOSE 8080
ENTRYPOINT ["/app/excelper-server"]
