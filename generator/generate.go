package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
)

func CurrentFile() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("can't determine self path")
	}
	return file, nil
}

var (
	packageName, defaultDomain, hostAPI,
	hostContent, hostNotify, sdkVersion,
	workDir,
	specVersion string
	apiVersion    int
	selfPath, dir string
)

func init() {
	var err error
	selfPath, err = CurrentFile()
	if err != nil {
		log.Fatalln(err)
	}
	dir = filepath.Dir(selfPath)
	workDir, err = os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
}

// TemplateData ...
type TemplateData struct {
	Package       string
	APIVersion    int    // 2
	DefaultDomain string //".dropboxapi.com"
	HostAPI       string // "api"
	HostContent   string // "content"
	HostNotify    string // "notify"
	SDKVersion    string // "UNKNOWN SDK VERSION"
	SpecVersion   string
}

func main() {
	app := cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "api-version",
				Destination: &apiVersion,
				Aliases:     []string{"v"},
				Value:       2,
			},
			&cli.StringFlag{
				Name:        "package",
				Destination: &packageName,
				Value:       "dropbox",
				Aliases:     []string{"p"},
			},
			&cli.StringFlag{
				Name:        "domain",
				Destination: &defaultDomain,
				Value:       "dropboxapi.com",
				Aliases:     []string{"d"},
			},
			&cli.StringFlag{
				Name:        "host-api",
				Destination: &hostAPI,
				Value:       "api",
				Aliases:     []string{"a"},
			},
			&cli.StringFlag{
				Name:        "host-content",
				Destination: &hostContent,
				Aliases:     []string{"c"},
				Value:       "content",
			},
			&cli.StringFlag{
				Name:        "host-notify",
				Destination: &hostNotify,
				Aliases:     []string{"n"},
				Value:       "notify",
			},
			&cli.StringFlag{
				Name:        "sdk-version",
				Destination: &sdkVersion,
				Aliases:     []string{"s"},
				Value:       "6.0.3",
			},
		},
		Action: func(c *cli.Context) error {
			outFolder := filepath.Join(dir, "..", "sdk", "dropbox")
			if _, err := os.Stat(outFolder); err != nil {
				if err = os.MkdirAll(outFolder, 0775); err != nil {
					return err
				}
			}

			apiModule := filepath.Join(dir, "..", ".git",
				"modules", "generator", "dropbox-api-spec")
			data, err := os.ReadFile(filepath.Join(apiModule, "HEAD"))
			if err != nil {
				return err
			}
			refSuffix := strings.TrimSpace(strings.Split(string(data), ":")[1])
			data, err = os.ReadFile(filepath.Join(apiModule, refSuffix))
			if err != nil {
				return err
			}
			specVersion = strings.TrimSpace(string(data))

			tmpl, err := template.ParseFiles(filepath.Join(dir, "dropbox", "sdk.gohtml"))
			if err != nil {
				return err
			}

			rootFile := filepath.Join(outFolder, "sdk.go")
			f, err := os.OpenFile(rootFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0664)
			if err != nil {
				return err
			}
			defer func() {
				_ = f.Close()
			}()

			err = tmpl.Execute(f, TemplateData{
				Package:       packageName,
				APIVersion:    apiVersion,
				DefaultDomain: defaultDomain,
				HostAPI:       hostAPI,
				HostContent:   hostContent,
				HostNotify:    hostNotify,
				SDKVersion:    sdkVersion,
				SpecVersion:   specVersion,
			})
			if err != nil {
				return err
			}
			_, err = f.Seek(0, io.SeekStart)
			if err != nil {
				return err
			}

			goRsrc := filepath.Join(dir, "lib", "go_rsrc")
			if _, err = os.Stat(goRsrc); err != nil {
				if err = os.MkdirAll(goRsrc, 0775); err != nil {
					return err
				}
			}
			f2, err := os.OpenFile(filepath.Join(goRsrc, "sdk.go"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0664)
			if err != nil {
				return err
			}
			if _, err = io.Copy(f2, f); err != nil {
				_ = f2.Close()
				return err
			}
			_ = f2.Close()

			outFile, err := os.OpenFile(filepath.Join(dir, "stone.out.log"),
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer func() {
				_ = outFile.Close()
			}()
			errFile, err := os.OpenFile(
				filepath.Join(dir, "stone.err.log"),
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer func() {
				_ = outFile.Close()
			}()

			matches, err := filepath.Glob(strings.Join(
				[]string{dir, "dropbox-api-spec", "*.stone"},
				string(filepath.Separator)))
			if err != nil {
				return err
			}

			com := exec.Command("stone", append([]string{
				"-v", "-a", ":all",
				filepath.Join(dir, "lib", "go_types.stoneg.py"),
				outFolder,
			}, matches...)...,
			)
			com.Stdout = outFile
			com.Stderr = errFile
			if err = com.Run(); err != nil {
				return err
			}
			com = exec.Command("stone", append([]string{
				"-v", "-a", ":all",
				filepath.Join(dir, "lib", "go_client.stoneg.py"),
				outFolder,
			}, matches...)...,
			)
			com.Stdout = outFile
			com.Stderr = errFile
			if err = com.Run(); err != nil {
				return err
			}

			tmpl, err = template.ParseFiles(filepath.Join(dir, "dropbox", "auth.gohtml"))
			if err != nil {
				return err
			}

			// auth workaround
			authFile, err := os.OpenFile(filepath.Join(outFolder, "auth", "sdk.go"),
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer func() {
				_ = authFile.Close()
			}()

			if err = tmpl.Execute(authFile, nil); err != nil {
				return err
			}

			tmpl, err = template.ParseFiles(filepath.Join(dir, "dropbox", "file_properties.gohtml"))
			if err != nil {
				return err
			}

			// auth workaround
			tagFile, err := os.OpenFile(filepath.Join(outFolder, "file_properties", "tagged.go"),
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer func() {
				_ = tagFile.Close()
			}()

			if err = tmpl.Execute(tagFile, nil); err != nil {
				return err
			}

			// append json iter to each file

			return exec.Command("goimports", "-l", "-w", outFolder).Run()
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
