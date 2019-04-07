package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gokyle/goconfig"
	"github.com/qbit/mycete/protector"
)

var c goconfig.ConfigMap
var err error
var temp_image_files_dir_ string
var feed2matrx_image_bytes_limit_ int64
var feed2matrx_image_count_limit_ int

func init() {
	var err error
	if feed2matrx_image_bytes_limit_, err = strconv.ParseInt(c.GetValueDefault("feed2matrix", "imagebyteslimit", "4194304"), 10, 64); err != nil {
		panic(err)
	}
	if feed2matrx_image_count_limit_, err = strconv.Atoi(c.GetValueDefault("feed2matrix", "imagecountlimit", "8")); err != nil {
		panic(err)
	}
}

/// Function Name Coding Standard
/// func runMyFunction    ... function that does not return and could be run a gorouting, e.g. go runMyFunction
/// func taskMyFunction  ... function that internally lauches a goroutine

func main() {
	cfile := flag.String("conf", "/etc/mycete.conf", "Configuration file")
	flag.Parse()

	protector.Protect("stdio rpath cpath wpath fattr inet dns")

	c, err = goconfig.ParseFile(*cfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if c.GetValueDefault("images", "enabled", "false") == "true" {
		temp_image_files_dir_, err = ioutil.TempDir(c.GetValueDefault("images", "temp_dir", "/tmp"), "mycete")
		if err != nil {
			panic(err)
		}
		if err = os.Chmod(temp_image_files_dir_, 0700); err != nil {
			panic(err)
		}
		defer os.RemoveAll(temp_image_files_dir_)
	}

	if c_charlimitstr, c_charlimitstr_set := c.GetValue("feed2matrix", "characterlimit"); c_charlimitstr_set && len(c_charlimitstr) > 0 {
		if charlimit, err := strconv.Atoi(c_charlimitstr); err == nil {
			matrix_notice_character_limit_ = charlimit
		}
	}

	go runMatrixPublishBot()

	///wait until Signal
	{
		ctrlc_c := make(chan os.Signal, 1)
		signal.Notify(ctrlc_c, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-ctrlc_c //block until ctrl+c is pressed || we receive SIGINT aka kill -1 || kill
		fmt.Println("Exiting")
	}
}
