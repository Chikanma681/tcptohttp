# Headers Package - Detailed Explanation

## Overview

This package implements HTTP header parsing and storage. It handles the parsing of raw HTTP header lines from a byte stream, storing them in a structured format, and providing methods to access and iterate over headers.

## Key Concepts

### HTTP Headers Format

HTTP headers come after the request line in an HTTP message. They look like:

```
Host: localhost:42069\r\n
User-Agent: curl/7.68.0\r\n
Accept: */*\r\n
\r\n
```

Each header line is terminated by `\r\n` (carriage return + line feed). An empty line (`\r\n` immediately after another `\r\n`) signals the end of headers.

---

## The Headers Struct

```go
type Headers struct {
    headers map[string]string
}
```

### Why Use a Struct Instead of a Map?

We could have defined `Headers` as `type Headers map[string]string`, but using a struct with an internal map gives us:

1. **Encapsulation**: We control how headers are accessed and modified
2. **Case-insensitive lookup**: HTTP header names are case-insensitive ("Content-Type" == "content-type"), but Go map keys are case-sensitive
3. **Potential for future features**: We could add validation, duplicate handling, or ordering later

The internal map `headers` stores header names as lowercase strings for consistent lookup.

---

## NewHeaders() - Constructor

```go
func NewHeaders() Headers {
    return Headers{
        headers: make(map[string]string),
    }
}
```

This creates a new `Headers` struct with an initialized (non-nil) map. In Go, you can't use an uninitialized map - it will panic if you try to write to it. `make(map[string]string)` creates an empty, usable map.

---

## Get() Method

```go
func (h Headers) Get(name string) string {
    return h.headers[strings.ToLower(name)]
}
```

### Key Points:

1. **Method receiver is `Headers` (value, not pointer)**: 
   - Since the struct only contains a map reference (which is internally a pointer), we don't need `*Headers`
   - Maps are reference types in Go - copying the struct copies the map reference, not the map contents

2. **Case-insensitive lookup**:
   - We convert the input `name` to lowercase before looking it up
   - This matches how HTTP specs define headers (case-insensitive)

3. **Returns empty string if not found**:
   - Accessing a non-existent key in a Go map returns the zero value (empty string for string type)

---

## Set() Method

```go
func (h Headers) Set(name, value string) {
    h.headers[strings.ToLower(name)] = value
}
```

Stores a header value, normalizing the name to lowercase for consistent lookup later.

---

## ForEach() Method - The Callback Pattern

```go
func (h Headers) ForEach(cb func(name, value string)) {
    for k, v := range h.headers {
        cb(k, v)
    }
}
```

### What is `cb`?

`cb` stands for "callback" - it's a function passed as an argument that `ForEach` will call for each header.

### Breaking Down the Signature:

```go
func(name, value string)
```

This means: "a function that takes two string parameters and returns nothing."

### How It Works:

1. You call `ForEach` and pass it a function
2. `ForEach` iterates over all headers in the internal map
3. For each header, it calls your function with the key and value

### Example Usage:

```go
headers.ForEach(func(name, value string) {
    fmt.Printf("%s: %s\n", name, value)
})
```

Or with a named function:

```go
func printHeader(name, value string) {
    fmt.Printf("%s: %s\n", name, value)
}

headers.ForEach(printHeader)
```

### Why Use Callbacks Instead of Returning a Slice?

1. **Memory efficiency**: No need to allocate a slice and copy all headers
2. **Lazy evaluation**: The caller controls what happens with each header
3. **Common pattern**: Used in Go standard library (e.g., `sync.Map.Range`, `flag.Visit`)

---

## parseHeader() - Low-level Parsing

```go
func parseHeader(fieldLine []byte) (string, string, error) {
    parts := bytes.SplitN(fieldLine, []byte(":"), 2)

    if len(parts) != 2 {
        return "", "", fmt.Errorf("malformed header")
    }

    name := parts[0]
    value := bytes.TrimSpace(parts[1])

    if bytes.HasSuffix(name, []byte(" ")) {
        return "", "", fmt.Errorf("malformed field name")
    }

    return string(name), string(value), nil
}
```

### Key Points:

1. **Uses `bytes.SplitN`, not `strings.SplitN`**:
   - We're working with `[]byte` (raw bytes) not `string`
   - More efficient for network data which arrives as bytes
   - `N=2` means split into at most 2 parts (name and value)

2. **Trims space from value only**:
   - `bytes.TrimSpace(parts[1])` removes leading/trailing whitespace from value
   - Does NOT trim the name - header names should have no spaces

3. **Validates name has no trailing space**:
   - Per HTTP spec, the colon must immediately follow the name
   - "Host : localhost" is invalid (space before colon)

---

## Parse() Method - The Main Entry Point

```go
func (h Headers) Parse(data []byte) (int, bool, error) {
    read := 0
    done := false

    for {
        idx := bytes.Index(data, rn)
        if idx == -1 {
            break
        }

        // EMPTY HEADER
        if idx == 0 {
            done = true
            read += len(rn)
            break
        }

        name, value, err := parseHeader(data[:idx])
        if err != nil {
            return 0, false, err
        }

        read += idx + len(rn)
        data = data[idx+len(rn):]
        h.headers[strings.ToLower(string(name))] = string(value)
    }

    return read, done, nil
}
```

### The Return Values Explained:

```go
func (h Headers) Parse(data []byte) (int, bool, error)
//                        bytes read, done?, error
```

1. **`read` (int)**: How many bytes were consumed from the input
   - The caller needs to know what data has been processed
   - Remaining data might contain the HTTP body or incomplete headers

2. **`done` (bool)**: Whether we've reached the end of headers
   - `true` when we encounter an empty line (`\r\n\r\n`)
   - The empty line signals the start of the body in HTTP/1.1

3. **`error`**: Any parsing error (malformed headers, etc.)

### The Loop Logic:

```
data: "Host: localhost\r\nUser-Agent: curl\r\n\r\nBody here"
       ^              ^
       idx of \r\n    consume this line
```

1. **Find `\r\n`**: Look for the line terminator
2. **Empty line check**: If `idx == 0`, the line is empty → headers are done
3. **Parse the line**: Extract name and value
4. **Update state**:
   - `read += idx + len(rn)` - account for bytes consumed
   - `data = data[idx+len(rn):]` - slide the window forward
   - Store the header (lowercasing the name)

### The Subtle Part - Slicing:

```go
data = data[idx+len(rn):]
```

This creates a new slice that points to the remaining unprocessed data. It doesn't copy the underlying array - just creates a new slice header. This is efficient O(1) operation.

---

## Token Validation (isToken and isValidTokenChar)

```go
func isValidTokenChar(ch byte) bool {
    // Quick check for alphanumeric
    if (ch >= 'A' && ch <= 'Z') ||
        (ch >= 'a' && ch <= 'z') ||
        (ch >= '0' && ch <= '9') {
        return true
    }

    // Check for allowed symbols
    switch ch {
    case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
        return true
    }

    return false
}

func isToken(str []byte) bool {
    for _, ch := range str {
        if !isValidTokenChar(ch) {
            return false
        }
    }
    return true
}
```

### What is a "Token"?

In HTTP specs (RFC 2616), header names must be "tokens" - specific allowed characters:
- Uppercase letters: A-Z
- Lowercase letters: a-z  
- Digits: 0-9
- Special characters: `!` `#` `$` `%` `&` `'` `*` `+` `-` `.` `^` `_` `` ` `` `|` `~`

### Why Validate?

Security and spec compliance. Malicious clients might send:
- `X-Header\r\n\r\nInjected-Header: value` (header injection)
- Non-printable characters that break parsing

These functions ensure header names contain only valid characters.

---

## Complete Data Flow Example

```
Raw TCP Data:
"GET /use-neovim-btw HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.68.0\r\n\r\n"

After Request.ParseRequestLine():
- Method: "GET"
- Target: "/use-neovim-btw"
- Version: "1.1"

Headers.Parse() is called with:
"Host: localhost:42069\r\nUser-Agent: curl/7.68.0\r\n\r\n"

First iteration:
- idx = bytes.Index(data, "\r\n") → 23
- data[:23] = "Host: localhost:42069"
- parseHeader() → ("Host", "localhost:42069", nil)
- Store: headers["host"] = "localhost:42069"
- read = 25 (23 + len("\r\n"))
- data = "User-Agent: curl/7.68.0\r\n\r\n"

Second iteration:
- idx = 25
- Store: headers["user-agent"] = "curl/7.68.0"
- read = 27 (25 + 2)
- data = "\r\n"

Third iteration:
- idx = 0 (empty line found!)
- done = true
- break

Return: (52, true, nil)

Final Headers map:
{
    "host": "localhost:42069",
    "user-agent": "curl/7.68.0"
}
```

---

## Summary

1. **Headers struct** wraps a map for encapsulation and case-insensitive lookup
2. **NewHeaders()** properly initializes the internal map
3. **Get()** normalizes keys to lowercase for consistent lookup
4. **Set()** stores values with lowercase keys
5. **ForEach()** uses a callback pattern for memory-efficient iteration
6. **Parse()** processes raw bytes, extracting headers line-by-line until empty line
7. **Token validation** ensures header names contain only valid characters per HTTP spec
