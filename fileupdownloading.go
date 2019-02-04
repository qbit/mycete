package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/matrix-org/gomatrix"
)

/// unfortunately, since neither go-twitter, anaconda or go-mastodon implement an io.Reader interface we have to use actual temporary files

//TODO: limit filesize
func readFileIntoBase64(filepath string) (string, error) {
	contents, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(contents), nil
}

//TODO: limit file size
//TODO: limit number of files stored in /run/..
func saveMatrixFile(cli *gomatrix.Client, nick, matrixurl string) error {
	if !strings.Contains(matrixurl, "mxc://") {
		return fmt.Errorf("saveMatrixFile: not a matrix content uri mxc://..")
	}
	matrixmediaurlpart := strings.Split(matrixurl, "mxc://")[1]
	imgfilepath := hashNickToPath(nick)
	imgtmpfilepath := imgfilepath + ".tmp"

	/// Create the file (implies truncate)
	fh, err := os.OpenFile(imgtmpfilepath, os.O_WRONLY|os.O_CREATE, 0400)
	if err != nil {
		return err
	}
	defer fh.Close()

	/// Download image
	mcxurl := cli.BuildBaseURL("/_matrix/media/r0/download/", matrixmediaurlpart)
	resp, err := http.Get(mcxurl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(fh, resp.Body)
	if err != nil {
		return err
	}
	os.Rename(imgtmpfilepath, imgfilepath)
	return nil
}

func rmFile(nick string) {
	// log.Println("removing file for", nick)
	os.Remove(hashNickToPath(nick))
}

/// return hex(sha256()) of string
/// used so malicous user can't use malicous filename that is defined by nick. (and hash collision or guessing not so big a threat here.)
func hashNickToPath(matrixnick string) string {
	shasum := make([]byte, sha256.Size)
	shasum32 := sha256.Sum256([]byte(matrixnick))
	copy(shasum[0:sha256.Size], shasum32[0:sha256.Size])
	return path.Join(temp_image_files_dir_, hex.EncodeToString(shasum))
}
