Go Image Server
===============

This program is used in order to distribute gridfs files fast with nginx. 

Compilation:
-----

* In the projects root directory export GOPATH=\`pwd\`
* run ./deps.sh to install depedencies
* go install voycer.com will install the binary in bin/voycer.com

Instructions
-----
start the server with:

    ./server --port=27017 --host=your.mongo.host

or use an init script. 

Nginx Configuration
-----

The configuration section for your media vhost could look something like this:

    location ^~ /media/ {
         proxy_set_header X-Real-IP $remote_addr;
         proxy_set_header X-Forwarded-For $remote_addr;
         proxy_set_header Host $http_host;
         proxy_pass http://127.0.0.1:8000/mongo_database/27017/;
    }
    
Cross compilation
-----
For cross compilation see http://dave.cheney.net/2012/09/08/an-introduction-to-cross-compilation-with-go
It should be enough to use the following commands:

* cd ~
* git clone git://github.com/davecheney/golang-crosscompile.git
* source golang-crosscompile/crosscompile.bash
* go-crosscompile-build-all

After this setup you can use the build-linux.sh to compile the image server for x64 linux servers.

