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


    location @image {
        proxy_pass http://127.0.0.1:8000/database/port/;
    }    

    location ^~ /media/ {
         proxy_set_header X-Real-IP $remote_addr;
         proxy_set_header X-Forwarded-For $remote_addr;
         proxy_set_header Host $http_host;
         try_files $uri @image;
    }