#!/bin/bash
bash minify.sh && go build -mod=vendor && sudo MONGO_URL=127.0.0.1:27017 ./newsfeedreader -env local
