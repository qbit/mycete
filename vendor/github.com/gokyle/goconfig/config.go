package goconfig

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
)

// ConfigMap is shorthand for the type used as a config struct.
type ConfigMap map[string]map[string]string

var (
	configSection    = regexp.MustCompile("^\\s*\\[\\s*(\\w+)\\s*\\]\\s*$")
	quotedConfigLine = regexp.MustCompile("^\\s*(\\w+)\\s*=\\s*[\"'](.*)[\"']\\s*$")
	configLine       = regexp.MustCompile("^\\s*(\\w+)\\s*=\\s*(.*)\\s*$")
	commentLine      = regexp.MustCompile("^#.*$")
	blankLine        = regexp.MustCompile("^\\s*$")
)

var DefaultSection = "default"

// ParseFile takes the filename as a string and returns a ConfigMap.
func ParseFile(fileName string) (cfg ConfigMap, err error) {
	var file *os.File
	file, err = os.Open(fileName)
	if err != nil {
		return
	}
	defer file.Close()
	return ParseReader(file)
}

func ParseReader(r io.Reader) (cfg ConfigMap, err error) {
	cfg = make(ConfigMap, 0)
	buf := bufio.NewReader(r)

	var (
		line           string
		longLine       bool
		currentSection string
		lineBytes      []byte
		isPrefix       bool
	)

	for {
		err = nil
		lineBytes, isPrefix, err = buf.ReadLine()
		if io.EOF == err {
			err = nil
			break
		} else if err != nil {
			break
		} else if isPrefix {
			line += string(lineBytes)

			longLine = true
			continue
		} else if longLine {
			line += string(lineBytes)
			longLine = false
		} else {
			line = string(lineBytes)
		}

		if commentLine.MatchString(line) {
			continue
		} else if blankLine.MatchString(line) {
			continue
		} else if configSection.MatchString(line) {
			section := configSection.ReplaceAllString(line,
				"$1")
			if section == "" {
				err = fmt.Errorf("invalid structure in file")
				break
			} else if !cfg.SectionInConfig(section) {
				cfg[section] = make(map[string]string, 0)
			}
			currentSection = section
		} else if configLine.MatchString(line) {
			regex := configLine
			if quotedConfigLine.MatchString(line) {
				regex = quotedConfigLine
			}
			if currentSection == "" {
				currentSection = DefaultSection
				if !cfg.SectionInConfig(currentSection) {
					cfg[currentSection] = make(map[string]string, 0)
				}
			}
			key := regex.ReplaceAllString(line, "$1")
			val := regex.ReplaceAllString(line, "$2")
			if key == "" {
				continue
			}
			cfg[currentSection][key] = val
		} else {
			err = fmt.Errorf("invalid config file")
			break
		}
	}
	return
}

// SectionInConfig determines whether a section is in the configuration.
func (c *ConfigMap) SectionInConfig(section string) bool {
	for s, _ := range *c {
		if section == s {
			return true
		}
	}
	return false
}

// ListSections returns the list of sections in the config map.
func (c *ConfigMap) ListSections() (sections []string) {
	for section, _ := range *c {
		sections = append(sections, section)
	}
	return
}

// WriteFile writes out the configuration to a file.
func (c *ConfigMap) WriteFile(filename string) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return
	}
	defer file.Close()

	for _, section := range c.ListSections() {
		sName := fmt.Sprintf("[ %s ]\n", section)
		_, err = file.Write([]byte(sName))
		if err != nil {
			return
		}

		for k, v := range (*c)[section] {
			line := fmt.Sprintf("%s = %s\n", k, v)
			_, err = file.Write([]byte(line))
			if err != nil {
				return
			}
		}
		_, err = file.Write([]byte{0x0a})
		if err != nil {
			return
		}
	}
	return
}

// AddSection creates a new section in the config map.
func (c *ConfigMap) AddSection(section string) {
	if nil != (*c)[section] {
		(*c)[section] = make(map[string]string, 0)
	}
}

// AddKeyVal adds a key value pair to a config map.
func (c *ConfigMap) AddKeyVal(section, key, val string) {
	if "" == section {
		section = DefaultSection
	}

	if nil == (*c)[section] {
		c.AddSection(section)
	}

	(*c)[section][key] = val
}

// Retrieve the value from a key map.
func (c *ConfigMap) GetValue(section, key string) (val string, present bool) {
	if c == nil {
		return
	}

	if section == "" {
		section = DefaultSection
	}

	cm := *c
	_, ok := cm[section]
	if !ok {
		return
	}

	val, present = cm[section][key]
	return
}

// Retrieve the value from a key map, or provide a default.
func (c *ConfigMap) GetValueDefault(section, key, value string) (val string) {
	kval, ok := c.GetValue(section, key)
	if !ok {
		return value
	}
	return kval
}

// Return a slice of strings containing all the keys in a section.
func (c *ConfigMap) SectionKeys(section string) (keys []string, present bool) {
	if c == nil {
		return nil, false
	}

	if section == "" {
		section = DefaultSection
	}

	cm := *c
	s, ok := cm[section]
	if !ok {
		return nil, false
	}

	keys = make([]string, 0, len(s))
	for key := range s {
		keys = append(keys, key)
	}

	return keys, true
}

// Base64 decodes a standard base64-encoded string.
func decBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// hex decodes a hex-encoded string.
func decHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

type Decoder func(string) ([]byte, error)

const (
	// Base64 indicates a base64-encoded string.
	Base64 int = iota + 1

	// Hex indicates a hex-encoded string.
	Hex
)

// Decoders contains a mapping of decoding functions.
var Decoders = map[int]Decoder{
	Base64: decBase64,
	Hex:    decHex,
}

// RegisterDecoder adds a new decoding function.
func RegisterDecoder(t int, decoder Decoder) {
	Decoders[t] = decoder
}

// ErrKeyNotPresent is returned from DecodeValue when no such key
// exists in the specified section.
var ErrKeyNotPresent = errors.New("goconfig: key not present")

// ErrDecoderUnavailable is returned when an invalid decoder is
// specified.
var ErrDecoderUnavailable = errors.New("goconfig: decoder unavailable")

// DecodeValue retrieves the value of a key and applies a decoding
// function to retrieve a byte slice.
func (c *ConfigMap) DecodeValue(section, key string, decoder int) ([]byte, error) {
	v, ok := c.GetValue(section, key)
	if !ok {
		return nil, ErrKeyNotPresent
	}

	dec, ok := Decoders[decoder]
	if !ok {
		return nil, ErrDecoderUnavailable
	}

	return dec(v)
}
