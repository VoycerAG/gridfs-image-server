package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//GridfsStorage will be made private
type GridfsStorage struct {
	Connection *mgo.Session
}

//Storage interface can be implemented
//to use the image server with any backend you like
type Storage interface {
	StoreChildImage(database, imageFormat string, imageData io.Reader, imageWidth, imageHeight int, original Cacheable, entry *Entry) (Cacheable, error)
	FindImageByParentID(namespace, id string, entry *Entry) (Cacheable, error)
	FindImageByParentFilename(namespace, filename string, entry *Entry) (Cacheable, error)
	IsValidID(id string) bool
}

//NewGridfsStorage returns a new gridfs storage provider
func NewGridfsStorage(con *mgo.Session) (Storage, error) {
	if con == nil {
		return nil, errors.New("mgo.Session must be set")
	}

	return &GridfsStorage{Connection: con}, nil
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

//IsValidID will return true if id is a valid bson object id
func (g GridfsStorage) IsValidID(id string) bool {
	return bson.IsObjectIdHex(id)
}

//FindImageByParentID returns either the resized image that actually exists, or the original if entry is nil
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
		return gridFileCacheable{mf: &mgo.GridFile{}}, fmt.Errorf("no image found for id %s", id)
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

func getRandomFilename(extension string) string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%s", time.Now().Nanosecond())))
	md := hash.Sum(nil)
	mdStr := hex.EncodeToString(md)

	return fmt.Sprintf("%s.%s", mdStr, extension)
}

//StoreChildImage will create a new image from source
func (g GridfsStorage) StoreChildImage(
	database,
	imageFormat string,
	reader io.Reader,
	imageWidth,
	imageHeight int,
	original Cacheable,
	entry *Entry,
) (Cacheable, error) {
	gridfs := g.Connection.DB(database).GridFS("fs")
	targetfile, err := gridfs.Create(getRandomFilename(imageFormat))

	if err != nil {
		return nil, err
	}

	defer targetfile.Close()

	_, err = io.Copy(targetfile, reader)

	if err != nil {
		targetfile.Abort()
		return nil, err
	}

	metadata := bson.M{
		"width":            imageWidth,
		"height":           imageHeight,
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

	targetfile.SetContentType("image/" + imageFormat)
	targetfile.SetMeta(metadata)

	return &gridFileCacheable{mf: targetfile}, nil
}
