FROM vimagick/alpine-arm:latest
RUN mkdir -p /app
WORKDIR /app
RUN cd /app
COPY newsfeedreader /app/newsfeedreader
COPY public /app/public
COPY manifest.json /app/manifest.json
RUN apk update && apk add tzdata
RUN cp /usr/share/zoneinfo/Europe/Helsinki /etc/localtime
RUN echo "Europe/Helsinki" > /etc/timezone
RUN apk del tzdata
EXPOSE 1300
ENV MONGO_URL 192.168.0.5:27017
CMD ["./newsfeedreader"]
