FROM golang:1.22 as build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /bin/app ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /bin/app /app
COPY web /web
COPY internal/db/schema.sql /internal/db/schema.sql
ENV DB_PATH=/data/app.db
ENV APP_SECRET=change-me
EXPOSE 8080
ENTRYPOINT ["/app"]
