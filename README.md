# sandbox
online judge sandbox

# Prepare
1. make directory : /tmp/snow
2. download python docker image : docker export $(docker create python:3-slim) | tar -C /tmp/snow -xzv -
3. download java docker image : docker export $(docker create azul/zulu-openjdk:17) | tar -C /tmp/snow -xzv -

# Reference
https://github.com/justice-oj/sandbox/