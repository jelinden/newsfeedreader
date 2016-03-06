FROM alpine:latest
COPY newsfeedreader /newsfeedreader
COPY public /public
COPY manifest.json /manifest.json
RUN apk add tzdata
RUN cp /usr/share/zoneinfo/Europe/Helsinki /etc/localtime
RUN echo "Europe/Helsinki" >  /etc/timezone
RUN apk del tzdata
EXPOSE 1300
ENV MONGO_URL 192.168.0.5:27017
ENTRYPOINT ["./newsfeedreader"]
