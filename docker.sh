CGO_ENABLED=0 GOOS=linux go build -a --installsuffix cgo --ldflags='-s' -o newsfeedreader
docker build -t default:newsfeedreader .
docker run -it -p 1300:1300 -d default:newsfeedreader
