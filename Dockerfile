FROM alpine:latest
COPY newsfeedreader /newsfeedreader
COPY public /public
COPY manifest.json /manifest.json
EXPOSE 1300
ENV MONGO_URL 192.168.0.5:27017
ENTRYPOINT ["./newsfeedreader"]
