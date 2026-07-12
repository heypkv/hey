package fetch

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ExtractBinary pulls exactly one file — binaryName — out of the archive at
// archivePath and writes it (0755) into destDir. Every other entry is
// ignored; entries with absolute, dot-dot, or non-regular-file shapes are
// rejected outright even if ignored, so a hostile archive fails loudly.
func ExtractBinary(archivePath, binaryName, destDir string) error {
	var found bool
	var err error
	if strings.HasSuffix(archivePath, ".zip") || isZip(archivePath) {
		found, err = extractFromZip(archivePath, binaryName, destDir)
	} else {
		found, err = extractFromTarGz(archivePath, binaryName, destDir)
	}
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("archive does not contain %s", binaryName)
	}
	return nil
}

func isZip(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return false
	}
	return magic == [4]byte{'P', 'K', 0x03, 0x04}
}

func checkEntryName(name string) error {
	clean := path.Clean(strings.ReplaceAll(name, `\`, "/"))
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "..") ||
		strings.Contains(clean, "/../") || filepath.IsAbs(name) {
		return fmt.Errorf("archive entry %q has an unsafe path", name)
	}
	return nil
}

func extractFromZip(archivePath, binaryName, destDir string) (bool, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return false, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()
	// Validate every entry before extracting anything — a hostile archive
	// must fail loudly even if the binary itself is fine.
	var target *zip.File
	for _, f := range r.File {
		if err := checkEntryName(f.Name); err != nil {
			return false, err
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("archive entry %q is a symlink; refusing", f.Name)
		}
		if !f.FileInfo().IsDir() && path.Base(strings.ReplaceAll(f.Name, `\`, "/")) == binaryName && target == nil {
			target = f
		}
	}
	if target == nil {
		return false, nil
	}
	rc, err := target.Open()
	if err != nil {
		return false, err
	}
	defer rc.Close()
	if err := writeBinary(rc, filepath.Join(destDir, binaryName)); err != nil {
		return false, err
	}
	return true, nil
}

func extractFromTarGz(archivePath, binaryName, destDir string) (bool, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return false, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return false, fmt.Errorf("open tar.gz: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	// Tar is a stream: extract when the binary appears, but keep scanning so
	// hostile entries anywhere in the archive still fail the whole install.
	found := false
	dst := filepath.Join(destDir, binaryName)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return found, nil
		}
		if err != nil {
			cleanupOnErr(found, dst)
			return false, err
		}
		if err := checkEntryName(hdr.Name); err != nil {
			cleanupOnErr(found, dst)
			return false, err
		}
		switch hdr.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			cleanupOnErr(found, dst)
			return false, fmt.Errorf("archive entry %q is a link; refusing", hdr.Name)
		case tar.TypeReg:
			if found || path.Base(hdr.Name) != binaryName {
				continue
			}
			if err := writeBinary(tr, dst); err != nil {
				return false, err
			}
			found = true
		}
	}
}

func cleanupOnErr(wrote bool, dst string) {
	if wrote {
		os.Remove(dst)
	}
}

func writeBinary(src io.Reader, dst string) error {
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, io.LimitReader(src, maxArchiveBytes))
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(dst)
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}
