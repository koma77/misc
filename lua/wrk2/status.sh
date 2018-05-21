docker run --net host --rm -v ~:/tmp/lua   1vlad/wrk2-docker -s /tmp/lua/post.lua -c 100  -t1 -R100 -d5s https://echo.site.com/
