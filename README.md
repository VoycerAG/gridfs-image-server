<a href='https://travis-ci.org/VoycerAG/gridfs-image-server'><img src='https://secure.travis-ci.org/VoycerAG/gridfs-image-server.png?branch=master'></a>

Go Image Server
===============

This program is used in order to distribute gridfs files fast with nginx.

It has resizing capabalities and stores resized images in the gridfs filesystem as children
of the original files. 

Run using docker
-----
Using the default configuration and no newrelic monitoring:
```docker run --link mongodb:mongodb -v -p 50000:8000 voycerag/gridfs_image_server```

Optional environment parameters:
- NEWRELIC_LICENSE the license identifier for your newrelic account
- MONGODB_CONNECTION if you do not use linking you can specify this env variable (e.g. 172.17.42.1:27017)

Volumes:
- /etc/image-server - you can mount this folder to use a custom configuration.json file.

Example command with all parameters:
```
  docker run -v /etc/configs:/etc/image-server -p 50000:8000 -e NEWRELIC_LICENSE=someIdentifier -e MONGODB_CONNECTION=172.17.42.1:27017 voycerag/gridfs_image_server
```

Compilation
-----

* install project using go get github.com/VoycerAG/gridfs-image-server

Instructions
-----
```
  -config string
      path to the configuration file (default "configuration.json")
  -host string
      the database host with an optional port, localhost would suffice (default "localhost:27017")
  -license string
      your newrelic license key in order to enable monitoring
  -port int
      the server port where we will serve images (default 8000)
```

Image Server Configuration
-----

See the configuration.json file for examples on how to configure entries for the image server.

Newrelic Monitoring
-----
Simply enter a valid license key on startup, and the image server will be monitored with the plugin GoRelic.
You can find it under Plugins in your account.

Nginx Configuration
-----

The configuration section for your media vhost could look something like this:

    location ^~ /media/ {
         proxy_set_header X-Real-IP $remote_addr;
         proxy_set_header X-Forwarded-For $remote_addr;
         proxy_set_header Host $http_host;
         proxy_pass http://127.0.0.1:8000/mongo_database/;
    }
    

Now resized images can be retrieved by calling /media/filename?size=entry where as the original image
is still available with /media/filename. 

If an invalid entry was requested, the image server will return the original image instead.

## Changelog

Changes in Version 3:

- A bug that made initial resized image do not sent cache headers is fixed.
- If neither the original file, nor the resize file could be found, instead of a status code 400
the server will now respond with a status code 404.
- If resizing fails, there won't be a status code 400 anymore, instead it will be a status code 500.
- Configurations without resize type are no longer supported. 
- image magick fallback is dead, therfore you should at least use go 1.4 now, 1.5 is recommened.
