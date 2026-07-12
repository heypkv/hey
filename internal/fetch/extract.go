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

// ExtractTree extracts every entry of the archive at archivePath into destDir,
// preserving the archive's directory structure and executable bits. It applies
// the same defenses as ExtractBinary — zip-slip paths, symlinks and hardlinks
// are rejected outright — but keeps the whole tree, which service packs need
// (a database ships bin/, lib/ and share/ together, not one file).
func ExtractTree(archivePath, destDir string) error {
	if strings.HasSuffix(archivePath, ".zip") || isZip(archivePath) {
		return extractTreeZip(archivePath, destDir)
	}
	return extractTreeTarGz(archivePath, destDir)
}

// safeJoin resolves name under destDir and confirms the result stays within
// destDir even after cleaning — a second line of defense behind checkEntryName.
func safeJoin(destDir, name string) (string, error) {
	if err := checkEntryName(name); err != nil {
		return "", err
	}
	clean := filepath.FromSlash(path.Clean(strings.ReplaceAll(name, `\`, "/")))
	dst := filepath.Join(destDir, clean)
	rel, err := filepath.Rel(destDir, dst)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry %q escapes the destination", name)
	}
	return dst, nil
}

func extractTreeZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()
	// Validate every entry up front so a hostile archive fails before we write
	// anything to disk.
	for _, f := range r.File {
		if _, err := safeJoin(destDir, f.Name); err != nil {
			return err
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive entry %q is a symlink; refusing", f.Name)
		}
	}
	for _, f := range r.File {
		dst, err := safeJoin(destDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = writeTreeFile(rc, dst, f.Mode())
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTreeTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("open tar.gz: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		dst, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("archive entry %q is a link; refusing", hdr.Name)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			if err := writeTreeFile(tr, dst, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		}
	}
}

// writeTreeFile writes src to dst, keeping only the executable bit from the
// archive (any exec bit → 0755, otherwise 0644) so a hostile mode can't grant
// setuid or world-write.
func writeTreeFile(src io.Reader, dst string, mode os.FileMode) error {
	perm := os.FileMode(0o644)
	if mode&0o111 != 0 {
		perm = 0o755
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
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
