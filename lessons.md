# Lessons Learned

## 2026-06-26: Architecture Migration (Browser → Chrome Cookie API)

### Durable Lessons

**1. `DefaultBaseDir` should be a var, not a func**

Package-level functions that return derived paths make testing difficult because
they cannot be reassigned. Convert to `var DefaultBaseDir = func() string { ... }`
to allow test overrides with temp directories.

**2. UUID generation must use crypto/rand**

A naive `hex[time.Now().UnixNano()%16]` in a tight loop produces **identical**
characters for all 32 positions because `UnixNano()` doesn't advance between
iterations. Always use `crypto/rand.Read()` for UUID v4 generation.

**3. Sentinel errors vs wrapped errors**

When creating errors with `fmt.Errorf("%w: ...", sentinelErr)`, test assertions
must use `errors.Is(err, sentinelErr)` not `err != sentinelErr`. The `%w` verb
wraps the sentinel; direct comparison fails.

**4. Platform-specific code needs build tags + stubs**

Windows DPAPI calls require `//go:build windows` build tags. Always provide
a companion `//go:build !windows` stub file to keep cross-platform compilation
working. Each platform-specific function must have exactly one implementation
per build tag group.

**5. Hardcoded path separators don't work cross-platform**

Test expectations using forward slashes break on Windows (`filepath.Join`
produces backslashes). Always use `filepath.Join` / `filepath.FromSlash` for
path construction in tests and production code that runs cross-platform.

**6. SSE line buffer needs explicit sizing**

`bufio.Scanner` has a 64KB default buffer. ChatGPT SSE responses can have
lines longer than this (especially with large code blocks). Always set
`scanner.Buffer()` with an adequate initial size and max.
