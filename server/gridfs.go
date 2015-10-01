package server

import (
	"fmt"
	"image"
	"io"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//GridfsStorage will be made private
type GridfsStorage struct {
	Connection *mgo.Session
}

//Cacheable is an interface for caching
type Cacheable interface {
	CacheIdentifier() string
	LastModified() time.Time
	Data() ReadSeekCloser
	Name() string
}

//ReadSeekCloser implements io.ReadCloser and io.Seeker
type ReadSeekCloser interface {
	io.ReadCloser
	io.Seeker
}

//MetaContainer is an optional interface that can be implemented
//to return optional metadata
type MetaContainer interface {
	Meta() map[string]interface{}
}

//Identity returns a unique identifer for its implementor
type Identity interface {
	ID() interface{}
}

type gridFileCacheable struct {
	mf *mgo.GridFile
}

func (gfc gridFileCacheable) LastModified() time.Time {
	return gfc.mf.UploadDate()
}

func (gfc gridFileCacheable) CacheIdentifier() string {
	return gfc.mf.MD5()
}

func (gfc gridFileCacheable) Name() string {
	return gfc.mf.Name()
}

func (gfc gridFileCacheable) Data() ReadSeekCloser {
	return gfc.mf
}

func (gfc gridFileCacheable) Meta() map[string]interface{} {
	originalMetadata := bson.M{}
	if err := gfc.mf.GetMeta(&originalMetadata); err != nil {
		return map[string]interface{}{}
	}

	return originalMetadata
}

//implement `Identity` interface
func (gfc gridFileCacheable) ID() interface{} {
	return gfc.mf.Id()
}

//FindImageByParentID blub
func (g GridfsStorage) FindImageByParentID(namespace, id string, entry *Entry) (Cacheable, error) {
	gridfs := g.Connection.DB(namespace).GridFS("fs")
	var fp *mgo.GridFile
	var query bson.M

	if entry == nil {
		query = bson.M{"_id": bson.ObjectIdHex(id)}
	} else {
		query = bson.M{
			"metadata.original.$id": bson.ObjectIdHex(id),
			"metadata.size":         fmt.Sprintf("%dx%d", entry.Width, entry.Height),
			"metadata.resizeType":   entry.Type}
	}

	iter := gridfs.Find(query).Iter()
	gridfs.OpenNext(iter, &fp)

	if fp == nil {
		return gridFileCacheable{mf: fp}, fmt.Errorf("no image found for id %s", id)
	}

	return &gridFileCacheable{mf: fp}, nil

}

// FindImageByParentFilename returns either the resized image that actually exists, or the original if entry is nil
func (g GridfsStorage) FindImageByParentFilename(namespace, filename string, entry *Entry) (Cacheable, error) {
	gridfs := g.Connection.DB(namespace).GridFS("fs")
	var fp *mgo.GridFile
	var query bson.M

	if entry == nil {
		query = bson.M{"filename": filename}
	} else {
		query = bson.M{
			"metadata.originalFilename": filename,
			"metadata.size":             fmt.Sprintf("%dx%d", entry.Width, entry.Height),
			"metadata.resizeType":       entry.Type}
	}

	iter := gridfs.Find(query).Iter()
	gridfs.OpenNext(iter, &fp)
	if fp == nil {
		return nil, fmt.Errorf("no image found for filename %s", filename)
	}

	return &gridFileCacheable{mf: fp}, nil
}

//NewImage will create a new image from source
func (g GridfsStorage) NewImage(
	namespace, filename, format string,
	source image.Image,
	original Cacheable,
	entry *Entry,
	meta map[string]interface{},
) (Cacheable, error) {
	gridfs := g.Connection.DB(namespace).GridFS("fs")
	targetfile, err := gridfs.Create(filename)

	if err != nil {
		return nil, err
	}

	defer targetfile.Close()

	err = EncodeImage(targetfile, source, format)
	if err != nil {
		return nil, err
	}

	width := source.Bounds().Dx()
	height := source.Bounds().Dy()

	metadata := bson.M{
		"width":            width,
		"height":           height,
		"originalFilename": original.Name(),
		"resizeType":       entry.Type,
		"size":             fmt.Sprintf("%dx%d", entry.Width, entry.Height)}

	if identifier, ok := original.(Identity); ok {
		metadata["original"] = mgo.DBRef{Collection: "fs.files", Id: identifier.ID()}
	}

	if metaContainer, ok := original.(MetaContainer); ok {
		parentMeta := metaContainer.Meta()
		for k, v := range parentMeta {
			if _, exists := metadata[k]; !exists {
				metadata[k] = v
			}
		}
	}

	targetfile.SetContentType("image/" + format)
	targetfile.SetMeta(metadata)

	return &gridFileCacheable{mf: targetfile}, nil
}
