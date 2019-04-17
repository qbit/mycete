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

	"github.com/btittelbach/cachetable"
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

func osGetLimitedNumElementsInDir(directory string) (int, error) {
	f, err := os.Open(directory)
	if err != nil {
		return 0, err
	}
	fileInfo, err := f.Readdir(feed2matrx_image_count_limit_ + 1)
	f.Close()
	if err != nil && err != io.EOF {
		return 0, err
	}
	return len(fileInfo), nil
}

func getUserFileList(nick string) ([]string, error) {
	userdir := hashNickToUserDir(nick)
	f, err := os.Open(userdir)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(feed2matrx_image_count_limit_)
	f.Close()
	if err != nil && err != io.EOF {
		return nil, err
	}
	fullnames := make([]string, len(names))
	for idx, filename := range names {
		fullnames[idx] = path.Join(userdir, filename)
	}
	return fullnames, nil
}

func saveMatrixFile(cli *gomatrix.Client, nick, eventid, matrixurl string) error {
	if !strings.Contains(matrixurl, "mxc://") {
		return fmt.Errorf("image url not a matrix content mxc://..  uri")
	}
	matrixmediaurlpart := strings.Split(matrixurl, "mxc://")[1]
	userdir, imgfilepath := hashNickAndEventIdToPath(nick, eventid)
	os.MkdirAll(userdir, 0700)
	imgtmpfilepath := imgfilepath + ".tmp"

	/// limit number of files per user
	numfiles, err := osGetLimitedNumElementsInDir(userdir)
	if err != nil {
		return err
	}
	if numfiles >= feed2matrx_image_count_limit_ {
		return fmt.Errorf("Too many files stored. %d is the limit.", feed2matrx_image_count_limit_)
	}

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

func rmFile(nick, eventid string) error {
	// log.Println("removing file for", nick)
	_, fpath := hashNickAndEventIdToPath(nick, eventid)
	return os.Remove(fpath)
}

func rmAllUserFiles(nick string) error {
	return os.RemoveAll(hashNickToUserDir(nick))
}

/// return hex(sha256()) of string
/// used so malicous user can't use malicous filename that is defined by nick. (and hash collision or guessing not so big a threat here.)
func hashNickToUserDir(matrixnick string) string {
	shasum := make([]byte, sha256.Size)
	shasum32 := sha256.Sum256([]byte(matrixnick))
	copy(shasum[0:sha256.Size], shasum32[0:sha256.Size])
	return path.Join(temp_image_files_dir_, hex.EncodeToString(shasum))
}

func hashNickAndEventIdToPath(matrixnick, eventid string) (string, string) {
	shasum := make([]byte, sha256.Size)
	shasum32 := sha256.Sum256([]byte(eventid))
	copy(shasum[0:sha256.Size], shasum32[0:sha256.Size])
	userdir := hashNickToUserDir(matrixnick)
	return userdir, path.Join(userdir, hex.EncodeToString(shasum))
}

type MxUploadedImageInfo struct {
	mxcurl        string
	mimetype      string
	contentlength int64
	err           error
}

type MxContentUrlFuture struct {
	imgurl          string
	future_mxcurl_c chan MxUploadedImageInfo
}

func matrixUploadLink(mxcli *gomatrix.Client, url string) (*gomatrix.RespMediaUpload, string, int64, error) {
	response, err := mxcli.Client.Get(url)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return nil, "", 0, err
	}
	mimetype := response.Header.Get("Content-Type")
	clength := response.ContentLength
	if clength > feed2matrx_image_bytes_limit_ {
		return nil, "", 0, fmt.Errorf("media's size exceeds imagebyteslimit: %d > %d", clength, feed2matrx_image_bytes_limit_)
	}
	rmu, err := mxcli.UploadToContentRepo(response.Body, mimetype, clength)
	return rmu, mimetype, clength, err
}

func taskUploadImageLinksToMatrix(mxcli *gomatrix.Client) chan<- MxContentUrlFuture {
	futures_chan := make(chan MxContentUrlFuture, 42)
	go func() {
		mx_link_store, err := cachetable.NewCacheTable(70, 9, false)
		if err != nil {
			panic(err)
		}
		for future := range futures_chan {
			resp := MxUploadedImageInfo{}
			if saveddata, inmap := mx_link_store.Get(future.imgurl); inmap {
				resp = saveddata.Value.(MxUploadedImageInfo)
			} else { // else upload it
				if resp_media_up, mimetype, clength, err := matrixUploadLink(mxcli, future.imgurl); err == nil {
					resp.mxcurl = resp_media_up.ContentURI
					resp.contentlength = clength
					resp.mimetype = mimetype
					resp.err = err
					mx_link_store.Set(future.imgurl, resp)
				} else {
					resp.err = err
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
