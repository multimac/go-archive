package tgzfs_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/concourse/go-archiver/tgzfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Compress", func() {
	var srcPath string
	var buffer *bytes.Buffer
	var compressErr error

	BeforeEach(func() {
		dir, err := ioutil.TempDir("", "archive-dir")
		Expect(err).NotTo(HaveOccurred())

		err = os.Mkdir(filepath.Join(dir, "outer-dir"), 0755)
		Expect(err).NotTo(HaveOccurred())

		err = os.Mkdir(filepath.Join(dir, "outer-dir", "inner-dir"), 0755)
		Expect(err).NotTo(HaveOccurred())

		innerFile, err := os.Create(filepath.Join(dir, "outer-dir", "inner-dir", "some-file"))
		Expect(err).NotTo(HaveOccurred())

		_, err = innerFile.Write([]byte("sup"))
		Expect(err).NotTo(HaveOccurred())

		err = os.Symlink("some-file", filepath.Join(dir, "outer-dir", "inner-dir", "some-symlink"))
		Expect(err).NotTo(HaveOccurred())

		srcPath = filepath.Join(dir, "outer-dir")
		buffer = new(bytes.Buffer)
	})

	JustBeforeEach(func() {
		compressErr = tgzfs.Compress(srcPath, buffer)
	})

	It("writes a .tar.gz stream to the writer", func() {
		Expect(compressErr).NotTo(HaveOccurred())

		gr, err := gzip.NewReader(buffer)
		Expect(err).NotTo(HaveOccurred())

		reader := tar.NewReader(gr)

		header, err := reader.Next()
		Expect(err).NotTo(HaveOccurred())
		Expect(header.Name).To(Equal("outer-dir/"))
		Expect(header.FileInfo().IsDir()).To(BeTrue())

		header, err = reader.Next()
		Expect(err).NotTo(HaveOccurred())
		Expect(header.Name).To(Equal("outer-dir/inner-dir/"))
		Expect(header.FileInfo().IsDir()).To(BeTrue())

		header, err = reader.Next()
		Expect(err).NotTo(HaveOccurred())
		Expect(header.Name).To(Equal("outer-dir/inner-dir/some-file"))
		Expect(header.FileInfo().IsDir()).To(BeFalse())

		contents, err := ioutil.ReadAll(reader)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(contents)).To(Equal("sup"))

		header, err = reader.Next()
		Expect(err).NotTo(HaveOccurred())
		Expect(header.Name).To(Equal("outer-dir/inner-dir/some-symlink"))
		Expect(header.FileInfo().Mode() & os.ModeSymlink).To(Equal(os.ModeSymlink))
		Expect(header.Linkname).To(Equal("some-file"))
	})

	Context("with a trailing slash", func() {
		BeforeEach(func() {
			srcPath = srcPath + "/"
		})

		It("archives the directory's contents", func() {
			Expect(compressErr).NotTo(HaveOccurred())

			gr, err := gzip.NewReader(buffer)
			Expect(err).NotTo(HaveOccurred())

			reader := tar.NewReader(gr)

			header, err := reader.Next()
			Expect(err).NotTo(HaveOccurred())
			Expect(header.Name).To(Equal("./"))
			Expect(header.FileInfo().IsDir()).To(BeTrue())

			header, err = reader.Next()
			Expect(err).NotTo(HaveOccurred())
			Expect(header.Name).To(Equal("inner-dir/"))
			Expect(header.FileInfo().IsDir()).To(BeTrue())

			header, err = reader.Next()
			Expect(err).NotTo(HaveOccurred())
			Expect(header.Name).To(Equal("inner-dir/some-file"))
			Expect(header.FileInfo().IsDir()).To(BeFalse())

			contents, err := ioutil.ReadAll(reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("sup"))

			header, err = reader.Next()
			Expect(err).NotTo(HaveOccurred())
			Expect(header.Name).To(Equal("inner-dir/some-symlink"))
			Expect(header.FileInfo().Mode() & os.ModeSymlink).To(Equal(os.ModeSymlink))
			Expect(header.Linkname).To(Equal("some-file"))
		})
	})

	Context("with a single file", func() {
		BeforeEach(func() {
			srcPath = filepath.Join(srcPath, "inner-dir", "some-file")
		})

		It("archives the single file at the root", func() {
			Expect(compressErr).NotTo(HaveOccurred())

			gr, err := gzip.NewReader(buffer)
			Expect(err).NotTo(HaveOccurred())

			reader := tar.NewReader(gr)

			header, err := reader.Next()
			Expect(err).NotTo(HaveOccurred())
			Expect(header.Name).To(Equal("some-file"))
			Expect(header.FileInfo().IsDir()).To(BeFalse())

			contents, err := ioutil.ReadAll(reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("sup"))
		})
	})

	Context("when there is no file at the given path", func() {
		BeforeEach(func() {
			srcPath = filepath.Join(srcPath, "barf")
		})

		It("returns an error", func() {
			Expect(compressErr).To(BeAssignableToTypeOf(&os.PathError{}))
		})
	})
})
