#!/bin/bash
bash minify.sh &&
sleep 2 && 
go build -mod=vendor && 
killall newsfeedreader
sleep 2 &&
export MONGO_URL="mongodb://$MONGO_USER:$MONGO_PASSWORD@192.168.0.1:27017/news"
nohup ./newsfeedreader > news.log 2>&1&
sleep 2
ps aux|grep news
