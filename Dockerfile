FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Europe/Helsinki /etc/localtime && \
    echo "Europe/Helsinki" > /etc/timezone
RUN mkdir -p /app
WORKDIR /app
COPY newsfeedreader /app/newsfeedreader
COPY public /app/public
COPY manifest.json /app/manifest.json
EXPOSE 1300
ENV MONGO_URL mongodb://127.0.0.1:27017/news
ENTRYPOINT ["/app/newsfeedreader"]
