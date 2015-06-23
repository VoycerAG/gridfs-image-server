<a href='https://travis-ci.org/VoycerAG/gridfs-image-server'><img src='https://secure.travis-ci.org/VoycerAG/gridfs-image-server.png?branch=master'></a>

Go Image Server
===============

This program is used in order to distribute gridfs files fast with nginx. 

Compilation:
-----

* install project using go get github.com/VoycerAG/gridfs-image-server

Instructions
-----
start the server with:

    Usage of ./bin/VoycerAG:
      -config="configuration.json": path to the configuration file
      -host="localhost:27017": the database host with an optional port, localhost would suffice
      -port=8000: the server port where we will serve images

or use an init script. 

Image Server Configuration
-----

See the configuration.json file for examples on how to configure entries for the image server.

Nginx Configuration
-----

The configuration section for your media vhost could look something like this:

    location ^~ /media/ {
         proxy_set_header X-Real-IP $remote_addr;
         proxy_set_header X-Forwarded-For $remote_addr;
         proxy_set_header Host $http_host;
         proxy_pass http://127.0.0.1:8000/mongo_database/;
    }
    

Now images can be retrieved by calling /media/filename?size=entry

