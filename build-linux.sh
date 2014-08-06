CROSS_COMPILE=~/golang-crosscompile/crosscompile.bash
TARGET_FILENAME=image-server.linux.x64

source $CROSS_COMPILE && go-linux-amd64 build && mv VoycerGo $TARGET_FILENAME && echo $TARGET_FILENAME " Successful created" 