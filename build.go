package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hexops/cmder"
	"github.com/hexops/zgo/internal/errors"
	"github.com/mholt/archiver/v4"
)

func init() {
	const usage = ``

	// Parse flags for our subcommand.
	flagSet := flag.NewFlagSet("build", flag.ExitOnError)

	// Handles calls to our subcommand.
	handler := func(args []string) error {
		ctx := context.Background()

		// Load zgo.toml configuration if present.
		var cfg Config
		if err := LoadConfig("zgo.toml", &cfg); err != nil {
			return errors.Wrap(err, "zgo: LoadConfig")
		}

		// If building for macOS, ensure the Xcode SDK license agreement is accepted.
		if targetGOOS() == "darwin" {
			if !cfg.AcceptXCodeLicense {
				fmt.Fprintf(os.Stderr, "zgo: macOS target requires a copy of the macOS Xcode SDK, which is distributed\n")
				fmt.Fprintf(os.Stderr, "     under the terms at https://www.apple.com/legal/sla/docs/xcode.pdf\n")
				fmt.Fprintf(os.Stderr, "\n")
				return errors.New("zgo: to accept set AcceptXCodeLicense=true in zgo.toml or ZGO_ACCEPT_XCODE_LICENSE=true")
			}
			if err := ensureXCodeSDK(cfg); err != nil {
				return errors.Wrap(err, "zgo")
			}
		}

		// Ensure we have the version of Zig specified.
		zigExe, err := ensureZigVersion(cfg)
		if err != nil {
			return errors.Wrap(err, "zgo")
		}
		fmt.Fprintf(os.Stderr, "zgo: building for %s (zig version=%s)\n", zigTargetTriple(), cfg.Version)

		// Set up CC/CXX for CGO cross compilation
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "zgo: export CC='%s'\n", goCC(cfg))
			fmt.Fprintf(os.Stderr, "zgo: export CXX='%s'\n", goCXX(cfg))
		}
		os.Setenv("CC", goCC(cfg))
		os.Setenv("CXX", goCXX(cfg))

		// If build.zig exists, perform `sig build` for the target. This enables Zig build
		// integration.
		// TODO: need a way to change this cmd line
		if _, err := os.Stat("build.zig"); err == nil {
			err := execf(ctx, os.Stderr, true, nil, "", zigExe, []string{"build", "-Dtarget=" + zigTargetTriple()}...)
			if err != nil {
				return errors.Wrap(err, "zgo")
			}
		}

		env := os.Environ()
		env = enforcePATH(env, filepath.Dir(zigExe))

		goBuildArgs := []string{"build"}
		goBuildArgs = append(goBuildArgs, goBuildFlags()...)
		hasLdFlags := false
		for i, arg := range args {
			if arg == "-ldflags" {
				goBuildArgs = append(goBuildArgs, "-ldflags")
				goBuildArgs = append(goBuildArgs, args[i+1]+" "+goBuildLdFlags())
				hasLdFlags = true
			}
		}
		if !hasLdFlags {
			goBuildArgs = append(goBuildArgs, "-ldflags")
			goBuildArgs = append(goBuildArgs, goBuildLdFlags())
		}
		err = execf(ctx, os.Stderr, true, env, "", "go", goBuildArgs...)
		if err != nil {
			return errors.Wrap(err, "zgo")
		}
		return nil
	}

	// Register the command.
	commands = append(commands, &cmder.Command{
		FlagSet: flagSet,
		Handler: handler,
		UsageFunc: func() {
			fmt.Fprintf(flag.CommandLine.Output(), "Usage of 'zgo %s':\n", flagSet.Name())
			flagSet.PrintDefaults()
			fmt.Fprintf(flag.CommandLine.Output(), "%s", usage)
		},
	})
}

func zigTargetTriple() string {
	return fmt.Sprintf("%s-%s%s", zigArch(targetGOARCH()), zigOS(targetGOOS()), zigSuffix(targetGOOS()))
}

func targetGOOS() string {
	goOS := os.Getenv("GOOS")
	if goOS == "" {
		goOS = runtime.GOOS
	}
	return goOS
}

func targetGOARCH() string {
	goArch := os.Getenv("GOARCH")
	if goArch == "" {
		goArch = runtime.GOARCH
	}
	return goArch
}

func zigSuffix(goOS string) string {
	switch goOS {
	case "windows":
		return "-gnu"
	case "linux":
		return "-musl"
	case "darwin":
		return ""
	default:
		panic("unsupported GOOS")
	}
}

func zigOS(goOS string) string {
	switch goOS {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	case "darwin":
		return "macos"
	default:
		panic("unsupported GOOS")
	}
}

func zigArch(goArch string) string {
	switch goArch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		panic("unsupported GOARCH")
	}
}

func goCC(cfg Config) string {
	return strings.Join([]string{"zig", "cc", "-target", zigTargetTriple(), goCCZigFlags(cfg)}, " ")
}

func goCCZigFlags(cfg Config) string {
	switch targetGOOS() {
	case "windows":
		return "-Wno-dll-attribute-on-redeclaration"
	case "linux":
		return ""
	case "darwin":
		root := filepath.Join(xcodeSDKDir(cfg), "root")
		frameworks := filepath.Join(root, "System/Library/Frameworks")
		return fmt.Sprintf("-F %s --sysroot %s", frameworks, root)
	default:
		panic("unhandled Zig OS (this is a bug, please report it)")
	}
}

func goCXX(cfg Config) string {
	return strings.Join([]string{"zig", "c++", "-target", zigTargetTriple(), goCXXZigFlags(cfg)}, " ")
}

func goCXXZigFlags(cfg Config) string {
	switch targetGOOS() {
	case "windows":
		return "-Wno-dll-attribute-on-redeclaration"
	case "linux":
		return ""
	case "darwin":
		root := filepath.Join(xcodeSDKDir(cfg), "root")
		frameworks := filepath.Join(root, "System/Library/Frameworks")
		return fmt.Sprintf("-F %s --sysroot %s", frameworks, root)
	default:
		panic("unhandled Zig OS (this is a bug, please report it)")
	}
}

func goBuildLdFlags() string {
	switch targetGOOS() {
	case "windows":
		return ""
	case "linux":
		return ""
	case "darwin":
		return "-s -w -linkmode external"
	default:
		panic("unhandled Zig OS (this is a bug, please report it)")
	}
}

func goBuildFlags() []string {
	switch targetGOOS() {
	case "windows":
		return nil
	case "linux":
		return nil
	case "darwin":
		return []string{"-buildmode=pie"}
	default:
		panic("unhandled Zig OS (this is a bug, please report it)")
	}
}

func ensureZigVersion(cfg Config) (string, error) {
	if cfg.Version == "system" {
		pathToZig, err := exec.LookPath("zig")
		if err != nil {
			return "", errors.New("zgo: zig is not installed (zgo is configured to use system Zig installation)")
		}
		fmt.Fprintf(os.Stderr, "zgo: configured to use system Zig installation (%s)\n", pathToZig)
		return "", nil
	}

	zigDir := filepath.Join(cfg.Dir, "zig", cfg.Version)
	_, err := os.Stat(zigDir)
	if os.IsNotExist(err) {
		archiveFileTmpPath := filepath.Join(cfg.Dir, "zig", "download.tmp")
		err := downloadExtractZig(cfg.Version, zigDir, archiveFileTmpPath)
		if err != nil {
			return "", err
		}
	}

	exeExt := ""
	hostGOOS := runtime.GOOS
	if hostGOOS == "windows" {
		exeExt = ".exe"
	}
	zigExe, _ := filepath.Abs(filepath.Join(zigDir, "zig"+exeExt))
	return zigExe, nil
}

func downloadExtractZig(version, dst, archiveFileTmpPath string) error {
	hostGOOS := runtime.GOOS
	hostGOARCH := runtime.GOARCH

	extension := "tar.xz"
	stripPathComponents := 1
	if hostGOOS == "windows" {
		extension = "zip"
		stripPathComponents = 0
	}
	archiveFileTmpPath = archiveFileTmpPath + "." + extension
	url := fmt.Sprintf("https://ziglang.org/builds/zig-%s-%s-%s.%s", zigOS(hostGOOS), zigArch(hostGOARCH), version, extension)
	_ = os.RemoveAll(archiveFileTmpPath)
	if err := os.MkdirAll(filepath.Dir(archiveFileTmpPath), os.ModePerm); err != nil {
		return errors.Wrap(err, "MkdirAll")
	}
	defer os.RemoveAll(archiveFileTmpPath)

	err := downloadFile(url, archiveFileTmpPath)
	if err != nil {
		return errors.Wrap(err, "download")
	}

	// Extract the Go archive
	err = extractArchive(archiveFileTmpPath, dst, stripPathComponents)
	if err != nil {
		return errors.Wrap(err, "extract")
	}
	return nil
}

func downloadFile(url string, filepath string) error {
	fmt.Fprintf(os.Stderr, "zgo: downloading: %s > %s\n", url, filepath)
	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrap(err, "Create")
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrap(err, "Copy")
	}
	return nil
}

func extractArchive(archiveFilePath, dst string, stripPathComponents int) error {
	fmt.Fprintf(os.Stderr, "zgo: extracting: %s > %s\n", archiveFilePath, dst)
	ctx := context.Background()
	handler := func(ctx context.Context, fi archiver.File) error {
		dstPath := filepath.Join(dst, stripComponents(fi.NameInArchive, stripPathComponents))
		if fi.IsDir() {
			err := os.MkdirAll(dstPath, os.ModePerm)
			return errors.Wrap(err, "MkdirAll")
		}

		src, err := fi.Open()
		if err != nil {
			return errors.Wrap(err, "Open")
		}
		defer src.Close()
		dst, err := os.Create(dstPath)
		if err != nil {
			return errors.Wrap(err, "Create")
		}
		_, err = io.Copy(dst, src)
		if err != nil {
			return errors.Wrap(err, "Copy")
		}
		err = os.Chmod(dstPath, fi.Mode().Perm())
		return errors.Wrap(err, "Chmod")
	}
	archiveFile, err := os.Open(archiveFilePath)
	if err != nil {
		return errors.Wrap(err, "Open(archiveFilePath)")
	}
	defer archiveFile.Close()

	type Format interface {
		Extract(
			ctx context.Context,
			sourceArchive io.Reader,
			pathsInArchive []string,
			handleFile archiver.FileHandler,
		) error
	}
	var format Format
	if strings.HasSuffix(archiveFilePath, ".tar.gz") {
		format = archiver.CompressedArchive{
			Compression: archiver.Gz{},
			Archival:    archiver.Tar{},
		}
	} else if strings.HasSuffix(archiveFilePath, ".tar.xz") {
		format = archiver.CompressedArchive{
			Compression: archiver.Xz{},
			Archival:    archiver.Tar{},
		}
	} else if strings.HasSuffix(archiveFilePath, ".zip") {
		format = archiver.Zip{}
	} else {
		return errors.New("unsupported archive format")
	}

	err = format.Extract(ctx, archiveFile, nil, handler)
	if err != nil {
		return errors.Wrap(err, "Extract")
	}
	return nil
}

func stripComponents(path string, n int) string {
	elems := strings.Split(path, string(os.PathSeparator))
	if len(elems) >= n {
		elems = elems[n:]
	}
	return strings.Join(elems, string(os.PathSeparator))
}

func xcodeSDKDir(cfg Config) string {
	d, _ := filepath.Abs(filepath.Join(cfg.Dir, "sdk-macos-12.0"))
	return d
}

func ensureXCodeSDK(cfg Config) error {
	sdkDir := filepath.Join(cfg.Dir, "sdk-macos-12.0") // so that it prints as relative
	return ensureCloned(
		"https://github.com/hexops/sdk-macos-12.0",
		"14613b4917c7059dad8f3789f55bb13a2548f83d",
		sdkDir,
	)
}

func ensureCloned(remoteURL, rev, dstDir string) error {
	ctx := context.Background()
	_, err := os.Stat(dstDir)
	if os.IsNotExist(err) {
		if err := execf(ctx, os.Stderr, true, nil, "", "git", "clone", "-c", "core.longpaths=true", remoteURL, dstDir); err != nil {
			return err
		}
	}

	var headBuf bytes.Buffer
	if err := execf(ctx, &headBuf, false, nil, dstDir, "git", "rev-parse", "HEAD"); err != nil {
		return err
	}
	head := strings.TrimSpace(headBuf.String())
	if head == rev {
		return nil
	}

	// Hard reset to the desired revision.
	if err := execf(ctx, os.Stderr, true, nil, dstDir, "git", "reset", "--hard", rev); err != nil {
		// If hard reset fails, try to fetch and then rerun.
		if err := execf(ctx, os.Stderr, true, nil, dstDir, "git", "fetch"); err != nil {
			return err
		}
		return execf(ctx, os.Stderr, true, nil, dstDir, "git", "reset", "--hard", rev)
	}
	return nil
}

func execf(ctx context.Context, f io.Writer, verbose bool, env []string, dir, name string, args ...string) error {
	if verbose {
		fmt.Fprintf(f, "zgo: $ %s\n", formatCmdLine(name, args...))
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = f
	cmd.Stdout = f
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, formatCmdLine(name, args...))
	}
	return nil
}

func formatCmdLine(name string, args ...string) string {
	cmdLine := make([]string, 0, 1+len(args))
	cmdLine = append(cmdLine, name)
	for _, arg := range args {
		if strings.Contains(arg, " ") {
			cmdLine = append(cmdLine, fmt.Sprintf(`'%s'`, arg))
			continue
		}
		cmdLine = append(cmdLine, arg)
	}
	return strings.Join(cmdLine, " ")
}

// enforcePath modifies and returns the env, with dir guaranteed to be first on the PATH.
func enforcePATH(env []string, dir string) []string {
	for i, v := range env {
		if !strings.HasPrefix(v, "PATH") {
			continue
		}

		list := strings.Split(strings.TrimPrefix(v, "PATH="), string(os.PathListSeparator))
		list = append([]string{dir}, list...)
		env[i] = "PATH=" + strings.Join(list, string(os.PathListSeparator))
	}
	return env
}
