# filesystem

## Features

This is a toy in-memory filesystem that supports basic operations. It supports the following:

- Thread-safe.
- Reading/writing files from the local filesystem.
- Get/Change current working directory.
- Creating new directories.
- Listing directory content.
- Creating/removing/moving files/dirs.
- Finding files/dirs matching a name and regex.

## Build

- Follow instructions to have Go installed on your system via [official instructions]
  (https://golang.org/doc/install).
- Go `cmd/filesystem` and run `go build .`.
- Run `./filesystem` and get instructions via `./filesystem -help`.

## Documentation

Full package documentation is available here for the [trie](https://pkg.go.dev/github.com/basharal/trie) and [filesystem](https://pkg.go.dev/github.com/basharal/filesystem/fs).

## Extensions

- Absolute/relative paths. All operations support both relative/absolute paths,
  which the exception of using `.` or `..`.
- Walking a subtree. Support walking subtrees (relative/absolute) and aborting
  upon finding the first match for regex.
- Streaming reads/writes. Single writer, multiple readers. Works even if the file
  is being moved since they use different locks.

## Design Choices

- Used a trie for file-system representation. Used an existing implementation, but
  added/customized functionality for our need. This [commit](https://github.com/basharal/trie/commit/fb543232634f87e369c01bcc765c041ae3320011) has the changes. The changes were
  made to support new operation and also to walk and stop at directory boundaries.
- This modified trie makes it efficient to do traversals both up and down via prefix
  matching.
- Currently, we lock the entire trie. A more efficient approach would be to lock
  subtrees, but it's more complicated since we need to guarantee lock-ordering for
  operations like move.

## Shortcuts

- Used recursive algorithm for walking the trie. Easier to implement. Shouldn't be
  be used for production since it can overflow the stack.
- Don't support moving where the destination is a directory, but source is a file.
  Both src/destination need to be files.
- Lots of cases that should be tested but didn't have time. There are probably bugs.
- Don't support spaces in paths in the command-line (filesystem supports them though)

## Possible Extensions

It's really easy to add new functionality, such as permissions...etc. There are clear
abstractions.

One thing that could be done is to have a distributed in-memory filesystem using this
filesystem.
