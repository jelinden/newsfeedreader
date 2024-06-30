#!/bin/bash
bash minify.sh && 
go build -mod=vendor && 
MONGO_URL="mongodb://$MONGO_USER:$MONGO_PASSWORD@192.168.0.1:27017/news" ./newsfeedreader
