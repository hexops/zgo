# zgo: Go + Zig = 💕

[Zig](https://ziglang.org) is an up-and-coming language that shares many similar qualities with Go. Many Go developers who encounter Zig find that Zig and Go appear to be best friends, and companies like Uber [use Zig](https://www.youtube.com/watch?v=SCj2J3HcEfc) as part of their Go build toolchain to ensure all of their developers have a seamless CGO/cross-compilation experience.

## Early stages

This is an early-stages MVP version. You can use it for cross-compilation, but may find issues. If you do, please open an issue and I'll be happy to look into it for you. :)

## CGO cross-compilation that Just Works™

Cross-compilation with CGO is usually pretty tough. Various projects try to solve this, (such as [purego](https://github.com/ebitengine/purego), which loads dynamic libraries at runtime but means you don't get a static library in the end, or [xgo](https://github.com/techknowlogick/xgo) which packages multiple gigabytes of C/C++ toolchains into a Docker image for you.)

Since Zig is not only a new language, but also a full C/C++ toolchain with cross-compilation that just works out of the box (similar to Go), in just <100 MB you can target any OS/architecture while building C/C++ code!

`zgo` makes using Zig for cross-compilation in your Go projects easy, by offerring a drop-in replacement for `go build` called `zgo build`, and by managing the Zig installation for you (in a `.zgo/zig` directory) so your developers don't need to think about installing Zig or having the right version.

### Usage

Normally in Go you cross compile using e.g.:

```
GOOS=linux GOARCH=amd64 go build
```

With zgo, you simply replace `go build` with `zgo build` and it'll cross-compile using Zig with full CGO support!

## Use Zig code from Go

Zig is a lower-level language than Go (a better C, so to speak.) Unlike Go, it doesn't feature a garbage collector, and has a much stronger emphasis on performance where every bit counts. Zig also has excellent C/C++ integration, and can build static and dynamic libraries callable from CGO. If you have particularly performance-sensitive code, consider writing it in Zig and calling it through CGO.

`zgo` aims to make integrating Zig codebases into Go codebases simpler. When you run `zgo build` it detects if a `build.zig` file is found in the working directory, and if so it will invoke `zig build -Dtarget=...` for you, so that your Zig code will have a chance to build. This means your Zig project can emit libraries to `zig-out/...` and then your Go code can simply use those via CGO!

## No magic

There is no magic here. One can just as easily run `zig build -Dtarget=...` first, followed by `go build` with the right Go linker flags, etc. `zgo build` doesn't replace your build system, it's just a minimal, opinionated, helpful wrapper. If you want to see what zgo does, simply set `ZGO_VERBOSE=true` to see the flags it passes to `go build` etc.

## Including your own CGO system dependencies

We're still working out best practices here with respect to zgo itself. Zig and CGO can obviously reference libraries/dependencies on your host system if you like, but as you are cross-compiling you will want them to be built for the target.

The general plan/guidance is to:

1. Write a `build.zig` for your dependency, so that Zig can build it as the C/C++ compiler and dump the resulting library and headers into `zig-out/...`
2. Use those from CGO.

Then, since `zgo` will invoke `zig build` for you it would "just work".

## Configuration

zgo can be configured in two ways: (1) via `zgo.toml` ([example](zgo.toml)) (2) via environment variables:

| `zgo.toml`                            | env                        | description                                                                                                                                                                                                               |
| ------------------------------------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `acceptXCodeLicense=false`            | `ZGO_ACCEPT_XCODE_LICENSE` | Whether you accept the [XCode license agreement](https://www.apple.com/legal/sla/docs/xcode.pdf), which must be accepted in order to target macOS.                                                                        |
| `version="0.11.0-dev.1615+f62e3b8c0"` | `ZGO_VERSION`              | The [Zig version](https://ziglang.org/download/) to use. If set to `system` the system Zig installation is used instead. If not set, the latest nightly is downloaded to the `.zgo/zig` directory and used in the future. |
| `verbose=false`                       | `ZGO_VERBOSE`              | enable verbose logging of what zgo does                                                                                                                                                                                   |
| `dir=".zgo"`                          | `ZGO_DIR`                  | The directory zgo should install Zig and fetch a ~160MB copy of the Xcode SDK into                                                                                                                                        |
