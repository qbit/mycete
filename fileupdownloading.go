package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/matrix-org/gomatrix"
)

/// unfortunately, since neither go-twitter, anaconda or go-mastodon implement an io.Reader interface we have to use actual temporary files

func checkImageBytesizeLimit(size int64) error {
	var max_image_bytes int64 = 10 * 1024 * 1024
	if c["server"]["twitter"] == "true" && size > imgbytes_limit_twitter_ {
		return fmt.Errorf("Image too large for Twitter. Please shrink to below %d bytes", imgbytes_limit_twitter_)
	}
	if c["server"]["mastodon"] == "true" && size > imgbytes_limit_mastodon_ {
		return fmt.Errorf("Image too large for Mastodon. Please shrink to below %d bytes", imgbytes_limit_mastodon_)
	}
	if size > max_image_bytes {
		return fmt.Errorf("Image is too large. Please shrink to below %d bytes", max_image_bytes)
	}
	return nil
}

func readFileIntoBase64(filepath string) (string, error) {
	contents, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(contents), nil
}

//TODO: limit number of files stored in /run/..
func saveMatrixFile(cli *gomatrix.Client, nick, matrixurl string) error {
	if !strings.Contains(matrixurl, "mxc://") {
		return fmt.Errorf("image url not a matrix content mxc://..  uri")
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

	// Check Filesize (again)
	if err = checkImageBytesizeLimit(resp.ContentLength); err != nil {
		os.Remove(imgtmpfilepath) //remove before close will work on unix/bsd. Not sure about windows, but meh.
		return err
	}

	// Write the body to file
	var bytes_written int64
	bytes_written, err = io.Copy(fh, resp.Body)
	if err != nil {
		return err
	}

	// Check Filesize (again)
	if err = checkImageBytesizeLimit(bytes_written); err != nil {
		if resp.ContentLength > 0 {
			log.Printf("Content-Length lied to us != bytes_written: %d != %d", resp.ContentLength, bytes_written)
		}
		os.Remove(imgtmpfilepath) //remove before close will work on unix/bsd. Not sure about windows, but meh.
		return err
	}

	os.Rename(imgtmpfilepath, imgfilepath)
	return nil
}

func rmFile(nick string) error {
	// log.Println("removing file for", nick)
	return os.Remove(hashNickToPath(nick))
}

/// return hex(sha256()) of string
/// used so malicous user can't use malicous filename that is defined by nick. (and hash collision or guessing not so big a threat here.)
func hashNickToPath(matrixnick string) string {
	shasum := make([]byte, sha256.Size)
	shasum32 := sha256.Sum256([]byte(matrixnick))
	copy(shasum[0:sha256.Size], shasum32[0:sha256.Size])
	return path.Join(temp_image_files_dir_, hex.EncodeToString(shasum))
}

type MxContentUrlFuture struct {
	imgurl          string
	future_mxcurl_c chan string
}

///TODO: use short fixed roundbuffer array instead of map. unlikely we will encounter the same imgurl twice in a long time.
// type MxContentStore struct {
// 	imgurl string
// 	mxcurl string
// }
/// TODO: don't be a memory hog
func taskUploadImageLinksToMatrix(mxcli *gomatrix.Client) chan<- MxContentUrlFuture {
	futures_chan := make(chan MxContentUrlFuture, 42)
	go func() {
		mx_link_store := make(map[string]string, 50)
		for future := range futures_chan {
			resp := ""
			if savedimg, inmap := mx_link_store[future.imgurl]; inmap {
				resp = savedimg
			} else { // else upload it
				if resp_media_up, err := mxcli.UploadLink(future.imgurl); err == nil {
					mx_link_store[future.imgurl] = resp_media_up.ContentURI
					resp = resp_media_up.ContentURI
				} else {
					log.Printf("uploadImageLinksToMatrix Error: url: %s, error: %s", future.imgurl, err.Error())
				}
			}
			//return something to future in every case
			if future.future_mxcurl_c != nil {
				select {
				case future.future_mxcurl_c <- resp:
				default:
				}
			}
		}
	}()
	return futures_chan
}
