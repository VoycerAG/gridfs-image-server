<a href='https://travis-ci.org/VoycerAG/gridfs-image-server'><img src='https://secure.travis-ci.org/VoycerAG/gridfs-image-server.png?branch=master'></a>

Go Image Server
===============

This program is used in order to distribute gridfs files fast with nginx. 

Compilation:
-----

* In the projects root directory export GOPATH=\`pwd\`
* run ./deps.sh to install depedencies
* go install github.com/VoycerAG will install the binary in bin/VoycerAG

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

Possible Problems
-----
Go 1.2 currently does not support reading of Interlaced PNG Files. Therefore, the Image Server
uses ImageMagick in the Background for the conversion of this special format. 
If your application uses Interlaced PNGs please install imagemagick ("covert x y") on the machine which uses the server.
Otherwise these PNG files will not be converted and result in 404 errors. 

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

Cross compilation
-----
For cross compilation see http://dave.cheney.net/2012/09/08/an-introduction-to-cross-compilation-with-go
It should be enough to use the following commands:

* cd ~
* git clone git://github.com/davecheney/golang-crosscompile.git
* source golang-crosscompile/crosscompile.bash
* go-crosscompile-build-all

After this setup you can use the build-linux.sh to compile the image server for x64 linux servers.

