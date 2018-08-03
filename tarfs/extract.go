package tarfs

import (
	"archive/tar"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func Extract(src io.Reader, dest string) error {
	if runtime.GOOS != "windows" {
		tarPath, err := exec.LookPath("tar")
		if err == nil {
			return tarExtract(tarPath, src, dest)
		}
	}

	tarReader := tar.NewReader(src)

	chown := os.Getuid() == 0

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if hdr.Name == "." {
			continue
		}

		err = ExtractEntry(hdr, dest, tarReader, chown)
		if err != nil {
			return err
		}
	}

	return nil
}

func ExtractEntry(header *tar.Header, dest string, input io.Reader, chown bool) error {
	filePath := filepath.Join(dest, header.Name)
	fileInfo := header.FileInfo()
	fileMode := fileInfo.Mode()

	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}

	switch header.Typeflag {
	case tar.TypeLink:
		err := os.Link(filepath.Join(dest, header.Linkname), filePath)
		if err != nil {
			return err
		}

		// skip chmod/chown
		return nil

	case tar.TypeSymlink:
		err := os.Symlink(header.Linkname, filePath)
		if err != nil {
			return err
		}

		// skip chmod/chown
		return nil

	case tar.TypeDir:
		err := os.MkdirAll(filePath, fileMode)
		if err != nil {
			return err
		}

	case tar.TypeReg:
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}

		_, err = io.Copy(file, input)
		if err != nil {
			return err
		}

		err = file.Close()
		if err != nil {
			return err
		}
	}

	err = os.Chmod(filePath, fileMode)
	if err != nil {
		return err
	}

	if chown {
		err = os.Chown(filePath, header.Uid, header.Gid)
		if err != nil {
			return err
		}
	}

	return nil
}
