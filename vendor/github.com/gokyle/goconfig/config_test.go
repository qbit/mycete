package goconfig

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
)

// FailWithError is a utility for dumping errors and failing the test.
func FailWithError(t *testing.T, err error) {
	fmt.Println("failed")
	if err != nil {
		fmt.Println("[!] ", err.Error())
	}
	t.FailNow()
}

// UnlinkIfExists removes a file if it exists.
func UnlinkIfExists(file string) {
	_, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		panic("failed to remove " + file)
	}
	os.Remove(file)
}

// stringSlicesEqual compares two string lists, checking that they
// contain the same elements.
func stringSlicesEqual(slice1, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	for i, _ := range slice1 {
		if slice1[i] != slice2[i] {
			return false
		}
	}

	for i, _ := range slice2 {
		if slice1[i] != slice2[i] {
			return false
		}
	}
	return true
}

func TestGoodConfig(t *testing.T) {
	testFile := "testdata/test.conf"
	fmt.Printf("[+] validating known-good config... ")
	cmap, err := ParseFile(testFile)
	if err != nil {
		FailWithError(t, err)
	} else if len(cmap) != 2 {
		FailWithError(t, err)
	}
	fmt.Println("ok")
}

func TestGoodConfig2(t *testing.T) {
	testFile := "testdata/test2.conf"
	fmt.Printf("[+] validating second known-good config... ")
	cmap, err := ParseFile(testFile)
	if err != nil {
		FailWithError(t, err)
	} else if len(cmap) != 1 {
		FailWithError(t, err)
	} else if len(cmap["default"]) != 3 {
		FailWithError(t, err)
	}
	fmt.Println("ok")
}

func TestBadConfig(t *testing.T) {
	testFile := "testdata/bad.conf"
	fmt.Printf("[+] ensure invalid config file fails... ")
	_, err := ParseFile(testFile)
	if err == nil {
		err = fmt.Errorf("invalid config file should fail")
		FailWithError(t, err)
	}
	fmt.Println("ok")
}

func TestWriteConfigFile(t *testing.T) {
	fmt.Printf("[+] ensure config file is written properly... ")
	const testFile = "testdata/test.conf"
	const testOut = "testdata/test.out"

	cmap, err := ParseFile(testFile)
	if err != nil {
		FailWithError(t, err)
	}

	defer UnlinkIfExists(testOut)
	err = cmap.WriteFile(testOut)
	if err != nil {
		FailWithError(t, err)
	}

	cmap2, err := ParseFile(testOut)
	if err != nil {
		FailWithError(t, err)
	}

	sectionList1 := cmap.ListSections()
	sectionList2 := cmap2.ListSections()
	sort.Strings(sectionList1)
	sort.Strings(sectionList2)
	if !stringSlicesEqual(sectionList1, sectionList2) {
		err = fmt.Errorf("section lists don't match")
		FailWithError(t, err)
	}

	for _, section := range sectionList1 {
		for _, k := range cmap[section] {
			if cmap[section][k] != cmap2[section][k] {
				err = fmt.Errorf("config key doesn't match")
				FailWithError(t, err)
			}
		}
	}
	fmt.Println("ok")
}

func TestQuotedValue(t *testing.T) {
	testFile := "testdata/test.conf"
	fmt.Printf("[+] validating quoted value... ")
	cmap, _ := ParseFile(testFile)
	val := cmap["sectionName"]["key4"]
	if val != " space at beginning and end " {
		FailWithError(t, errors.New("Wrong value in double quotes ["+val+"]"))
	}

	val = cmap["sectionName"]["key5"]
	if val != " is quoted with single quotes " {
		FailWithError(t, errors.New("Wrong value in single quotes ["+val+"]"))
	}
	fmt.Println("ok")
}
