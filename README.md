# sandbox
online judge sandbox

# Prepare
1. create memory cgroup : sudo cgcreate -a user:user -t user:user -g memory:snowbox
2. make directory : /tmp/snowbox
3. download python docker image : docker export $(docker create python:3-slim) | tar -C /tmp/snowbox -xzv -
4. download and decompress java to /usr/lib/jvm/zulu11
