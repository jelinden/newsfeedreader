#!/bin/bash
bash minify.sh && go build -mod=vendor && sudo MONGO_URL=192.168.0.5:27017 ./newsfeedreader -env local
