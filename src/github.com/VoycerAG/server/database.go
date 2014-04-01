package server

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

// FindImageByParentFilename returns either the resized image that actually exists, or the original if entry is nil
func FindImageByParentFilename(filename string, entry *Entry, gridfs *mgo.GridFS) (*mgo.GridFile, error) {
	var fp *mgo.GridFile
	var query bson.M

	if entry == nil {
		query = bson.M{"filename": filename}
	} else {
		query = bson.M{
			"metadata.originalFilename": filename,
			"metadata.width":            entry.Width,
			"metadata.resizeType":       entry.Type,
			"metadata.height":           entry.Height}
	}

	iter := gridfs.Find(query).Iter()
	gridfs.OpenNext(iter, &fp)

	if fp == nil {
		return fp, fmt.Errorf("no image found for %s", filename)
	}

	return fp, nil
}
