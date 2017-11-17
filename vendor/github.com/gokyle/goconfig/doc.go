/*
   Copyright (c) 2012 Kyle Isom <kyle@tyrfingr.is>

   Permission to use, copy, modify, and distribute this software for any
   purpose with or without fee is hereby granted, provided that the
   above copyright notice and this permission notice appear in all
   copies.

   THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL
   WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED
   WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE
   AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL
   DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA
   OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER
   TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
   PERFORMANCE OF THIS SOFTWARE.
*/

/*
   package goconfig provides parsing for simple configuration file.

   A configuration is expected to be in the format

        [ sectionName ]
        key1 = some value
        key2 = some other value
        # we want to explain the importance and great forethought
        # in this next value.
        key3 = unintuitive value
        [ anotherSection ]
        key1 = a value
        key2 = yet another value
        ...

   Blank lines are skipped, and lines beginning with # are considered
   comments to be skipped. It is an error to have a section marker ('[]')
   without a section name; however, if the keys and values are defined
   outside of a section, they are placed in the default section called
   "default".

   Parsing a file can be done with the ParseFile function. The sections
   can be retrived with the ListSection method, and the ListKeys method
   can be used to retrieve a list of keys. A value can be retrieved using
   the GetValue method, which returns a string containing the value of
   the key and a boolean that is true if the given key exists.

   Creating a new configuration and writing it to a file is also
   possible. For example, to create the previous example:

        cmap := new(goconfig.ConfigMap)

        cmap.AddSection("sectionName")
        cmap.AddKeyVal("sectionName", "key1", "some value")
        cmap.AddKeyVal("sectionName", "key2", "some other value")
        cmap.AddKeyVal("sectionName", "key3", "unintuitive value")

        cmap.AddSection("anotherSection")
        cmap.AddKeyVal("anotherSection", "key1", "a value")
        cmap.AddKeyVal("anotherSection", "key1", "yet another value")

        err := cmap.WriteFile("example.conf")
        if err != nil {
                log.Fatal("couldn't write config file: ", err.Error())
        }

   Note that attempting to add sections or key / value pairs to a
   nil ConfigMap will most likely result in panics.
*/
package goconfig
