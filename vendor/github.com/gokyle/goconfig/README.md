goconfig provides parsing for simple configuration file.

A configuration is expected to be in the format

```
[ sectionName ]
key1 = some value
key2 = some other value
# we want to explain the importance and great forethought
# in this next value.
key3 = unintuitive value
[ anotherSection ]
key1 = a value
key2 = yet another value
key3 = " This value is quoted as we want to begin with a space."
#...
```

Blank lines are skipped, and lines beginning with `#` are considered
comments to be skipped. It is an error to have a section marker ('[]')
without a section name. `key = ` lines will set the line to a blank
value. If no section is given, the default section (`default`).

If you want the value to start or end with spaces, you may quote
the value, in 'single' or "double" quotes.

Parsing a file can be done with the ParseFile function. It will return
a `map[string]map[string]string`. For example, if the section `foo` is
defined, and `foo = bar` is specified:

```
import "github.com/gokyle/goconfig"

func getFoo() string {
        conf, err := goconfig.ParseFile("config.conf")
	// error handling elided
        return conf["foo"]["bar"]
}
```

