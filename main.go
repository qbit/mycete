package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/gokyle/goconfig"
	"suah.dev/protect"
)

/// Configuration Globals
var (
	c                              goconfig.ConfigMap
	temp_image_files_dir_          string
	feed2matrx_image_bytes_limit_  int64
	feed2matrx_image_count_limit_  int
	matrix_notice_character_limit_ int = 1000
	guard_prefix_                  string
	reblog_cmd_                    string
	favourite_cmd_                 string
)

/// Function Name Coding Standard
/// func runMyFunction    ... function that does not return and could be run a gorouting, e.g. go runMyFunction
/// func taskMyFunction  ... function that internally lauches a goroutine

func mainWithDefers() {
	var err error
	//// Create image temp dir if needed
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

	///////////////////////////////////////////////////////////
	//// Start Bot and all Sub-Go-Routines
	go runMatrixPublishBot()

	///////////////////////////////////////////////////////////
	//// wait until Signal, then quit
	{
		ctrlc_c := make(chan os.Signal, 1)
		signal.Notify(ctrlc_c, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-ctrlc_c //block until ctrl+c is pressed || we receive SIGINT aka kill -1 || kill
	}
}

func main() {
	var err error

	cfile := flag.String("conf", "/etc/mycete.conf", "Configuration file")
	flag.Parse()

	_ = protect.Pledge("stdio rpath cpath wpath fattr inet dns")

	c, err = goconfig.ParseFile(*cfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	///////////////////////////////////////////////////////////
	//// pre-read and initialize gloabl configuration variables

	if c_charlimitstr, c_charlimitstr_set := c.GetValue("feed2matrix", "characterlimit"); c_charlimitstr_set && len(c_charlimitstr) > 0 {
		if charlimit, err := strconv.Atoi(c_charlimitstr); err == nil {
			matrix_notice_character_limit_ = charlimit
		}
	}

	if feed2matrx_image_bytes_limit_, err = strconv.ParseInt(c.GetValueDefault("feed2matrix", "imagebyteslimit", "4194304"), 10, 64); err != nil {
		panic(err)
	}
	if feed2matrx_image_count_limit_, err = strconv.Atoi(c.GetValueDefault("feed2matrix", "imagecountlimit", "4")); err != nil {
		panic(err)
	}

	guard_prefix_ = strings.TrimSpace(c.GetValueDefault("matrix", "guard_prefix", "t>"))
	reblog_cmd_ = strings.TrimSpace(c.GetValueDefault("matrix", "reblog_cmd", "reblog>"))
	favourite_cmd_ = strings.TrimSpace(c.GetValueDefault("matrix", "favourite_cmd", "+1>"))
	if guard_prefix_ == reblog_cmd_ || reblog_cmd_ == favourite_cmd_ || favourite_cmd_ == guard_prefix_ {
		panic("ERROR: guard_prefix, reblog_cmd or favourite_cmd MUST differ")
	} //https://chaos.social/@realraum/101880653017828628

	////////////////////////////////////////////////////////////
	//// run main Main where a defer will still be called before we exit
	mainWithDefers()
	fmt.Println("Exiting")
}
