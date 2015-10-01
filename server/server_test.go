package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	. "github.com/VoycerAG/gridfs-image-server/server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	testConfig = `
{
	"allowedEntries" : [
		{
			"name" : "45x35",
			"width" : 45,
			"height" : 35,
			"type" : "resize"
		},
		{
			"name" : "50x40",
			"width" : 50,
			"height" : 40,
			"type" : "crop"
		},
		{
			"name" : "50x50",
			"width" : 50,
			"height" : 50,
			"type" : "crop"
		},
		{
			"name" : "130x260",
			"width" : 130,
			"height" : 260,
			"type" : "fit"
		},
		{
			"name" : "302x302",
			"width" : 302,
			"height" : 302,
			"type" : "fit"
		}
	]
}		
	`
)

var _ = Describe("Server", func() {
	loadFixtureFile := func(source, target string, gridfs *mgo.GridFS, metadata map[string]string) error {
		fp, err := os.Open(source)
		if err != nil {
			return err
		}

		gf, err := gridfs.Create(target)
		if err != nil {
			return err
		}

		gf.SetMeta(metadata)

		defer gf.Close()
		_, err = io.Copy(gf, fp)

		return err
	}

	Context("Test basic responses", func() {
		var (
			rec          *httptest.ResponseRecorder
			config       *Config
			imageServer  Server
			connection   *mgo.Session
			database     *mgo.Database
			databaseName string
			gridfs       *mgo.GridFS
		)

		BeforeSuite(func() {
			var err error
			databaseName = "testdb"
			config, err = NewConfigFromBytes([]byte(testConfig))
			Expect(err).ToNot(HaveOccurred())
			connection, err = mgo.Dial("localhost:27017")
			connection.SetMode(mgo.Monotonic, true)
			Expect(err).ToNot(HaveOccurred())
			imageServer = NewImageServer(config, connection)
			database = connection.DB(databaseName)
			Expect(database).ToNot(BeNil())
			gridfs = database.GridFS("fs")
		})

		BeforeEach(func() {
			rec = httptest.NewRecorder()
		})

		It("Should response with welcome on /", func() {
			req, err := http.NewRequest("GET", "/", nil)
			Expect(err).ToNot(HaveOccurred())
			handler := imageServer.Handler()
			handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.Bytes()).To(ContainSubstring("Image Server."))
		})

		It("will response with 404 if image not found", func() {
			req, err := http.NewRequest("GET", "/invalid_testdatbase/notfound.jpg", nil)
			Expect(err).ToNot(HaveOccurred())
			handler := imageServer.Handler()
			handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})

		It("will deliver the original image without filter", func() {
			err := loadFixtureFile("../testdata/image.jpg", "test.jpg", gridfs, map[string]string{})
			Expect(err).ToNot(HaveOccurred())
			req, err := http.NewRequest("GET", "/"+databaseName+"/test.jpg", nil)
			Expect(err).ToNot(HaveOccurred())
			handler := imageServer.Handler()
			handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(len(rec.Body.Bytes())).To(BeNumerically(">", 0))
			Expect(rec.Header().Get("Etag")).ToNot(Equal(""))
		})

		It("will deliver the resized image with filter", func() {
			err := loadFixtureFile("../testdata/image.jpg", "test.jpg", gridfs, map[string]string{})
			Expect(err).ToNot(HaveOccurred())
			req, err := http.NewRequest("GET", "/"+databaseName+"/test.jpg?size=45x35", nil)
			Expect(err).ToNot(HaveOccurred())
			handler := imageServer.Handler()
			handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(len(rec.Body.Bytes())).To(BeNumerically(">", 0))
			Expect(rec.Header().Get("Etag")).ToNot(Equal(""))
		})

		It("will have original metadata entries after resize", func() {
			metadata := map[string]string{
				"copyright": "ACME Fantasia",
				"license":   "MIT",
			}

			err := loadFixtureFile("../testdata/image.jpg", "metadata.jpg", gridfs, metadata)
			Expect(err).ToNot(HaveOccurred())
			req, err := http.NewRequest("GET", "/"+databaseName+"/metadata.jpg?size=45x35", nil)
			Expect(err).ToNot(HaveOccurred())
			handler := imageServer.Handler()
			handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(len(rec.Body.Bytes())).To(BeNumerically(">", 0))
			Expect(rec.Header().Get("Etag")).ToNot(Equal(""))
			query := gridfs.Find(bson.M{
				"metadata.originalFilename": "metadata.jpg",
				"metadata.size":             "45x35",
				"metadata.resizeType":       "resize",
			})

			var file *mgo.GridFile
			ok := gridfs.OpenNext(query.Iter(), &file)
			Expect(ok).To(Equal(true), "could find file successfully")

			actual := map[string]string{}
			err = file.GetMeta(&actual)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(ContainElement("ACME Fantasia"))
			Expect(actual).To(ContainElement("45x35"))
			Expect(actual).To(ContainElement("resize"))
			Expect(actual).To(ContainElement("metadata.jpg"))
			Expect(actual).To(ContainElement("MIT"))
		})

		It("will respond only with not modified if correct if none match got sent", func() {
			err := loadFixtureFile("../testdata/image.jpg", "test.jpg", gridfs, map[string]string{})
			Expect(err).ToNot(HaveOccurred())
			req, err := http.NewRequest("GET", "/"+databaseName+"/test.jpg?size=45x35", nil)
			Expect(err).ToNot(HaveOccurred())
			query := gridfs.Find(bson.M{
				"metadata.originalFilename": "test.jpg",
				"metadata.size":             "45x35",
				"metadata.resizeType":       "resize",
			})

			var file *mgo.GridFile
			ok := gridfs.OpenNext(query.Iter(), &file)
			defer query.Iter().Close()
			Expect(ok).To(Equal(true), "could find file successfully")
			Expect(file.MD5()).To(Equal("f7e9e8e583180dd945da1b3f5acfa758"))
			req.Header.Set("If-None-Match", file.MD5())
			handler := imageServer.Handler()
			handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusNotModified))
			Expect(len(rec.Body.Bytes())).To(Equal(0))
		})

		AfterSuite(func() {
			database.DropDatabase()
		})
	})
})
