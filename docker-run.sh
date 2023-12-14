#!/bin/zsh
docker build -t gonode .
docker run -p 12000:12000 --privileged=true gonode
