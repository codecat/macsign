package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName(".macsign")
	viper.SetConfigType("toml")

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")

	if err := viper.ReadInConfig(); err != nil {
		log.Error("Unable to read config", "err", err)
		os.Exit(1)

		log.Info("Missing .macsign.toml. Creating a template for you now.")
		log.Info("You should edit this file before re-running!")
		log.Info("Tip: You can also put this file in ~/.macsign.toml.")

		f, err := os.Create(".macsign.toml")
		if err != nil {
			log.Error("Unable to create .macsign.toml", "err", err)
			os.Exit(1)
			return
		}

		f.WriteString(`[keychain]
profile = ""

[keychain.identity]
application = "Developer ID Application: "
installer = "Developer ID Installer: "
`)
		f.Close()

		os.Exit(1)
		return
	}

	keychainProfile := viper.GetString("keychain.profile")
	if keychainProfile == "" {
		log.Fatal("Missing keychain.profile in configuration! Did you forget to change the auto-generated macsign.toml?")
		return
	}

	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Info("Usage: macsign <paths>")
		os.Exit(1)
	}

	log.Info("MacSign starting", "keychain-profile", keychainProfile)

	// Make sure all paths exist
	for _, path := range flag.Args() {
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			log.Error("File does not exist: ", "path", path)
			os.Exit(1)
		}
	}

	// Codesigning
	for _, path := range flag.Args() {
		typeName := "application"
		codesignExe := "codesign"
		codesignArgs := make([]string, 0)

		if strings.HasSuffix(path, ".pkg") {
			typeName = "installer"
			codesignExe = "productsign"
			codesignArgs = append(codesignArgs, "--timestamp")
			codesignArgs = append(codesignArgs, "--sign", viper.GetString("keychain.identity.installer"))
			codesignArgs = append(codesignArgs, path, "__Signed_"+path)
		} else {
			codesignArgs = append(codesignArgs, "-s", viper.GetString("keychain.identity.application"))
			codesignArgs = append(codesignArgs, "-f")
			codesignArgs = append(codesignArgs, "--timestamp")
			codesignArgs = append(codesignArgs, path)
		}

		log.Info("Signing "+typeName, "path", path)

		output, err := exec.Command(codesignExe, codesignArgs...).CombinedOutput()
		if err != nil {
			log.Error("Unable to codesign "+typeName, "path", path, "err", err, "output", string(output))
			os.Exit(1)
		}

		// For installers, productsign output must be different than input.
		// So we must remove input and rename output.
		if strings.HasSuffix(path, ".pkg") {
			if err := os.Remove(path); err != nil {
				log.Error("Unable to remove unsigned package", "path", path, "err", err)
				os.Exit(1)
			}

			if err := os.Rename("__Signed_"+path, path); err != nil {
				log.Error("Unable to rename signed package", "path", "__Signed_"+path, "err", err)
				os.Exit(1)
			}
		}
	}

	// Zip for notarization
	zipFileName := fmt.Sprintf("__MacSign_%d.zip", time.Now().Unix())
	zipArgs := []string{"-r", zipFileName}
	zipArgs = append(zipArgs, flag.Args()...)
	output, err := exec.Command("zip", zipArgs...).CombinedOutput()
	if err != nil {
		log.Error("Unable to zip application for notarization", "path", zipFileName, "err", err, "output", string(output))
		os.Exit(1)
	}

	// Send zip to Apple for notarization
	log.Info("Notarizing with Apple, this may take a bit..", "paths", len(flag.Args()))
	output, err = exec.Command("xcrun", "notarytool", "submit", "--keychain-profile", keychainProfile, "--wait", zipFileName).CombinedOutput()
	if err != nil {
		log.Error("Notarization failed", "err", err, "output", string(output))
		os.Exit(1)
	}
	log.Info("Notarizing successful!")

	// Delete zip
	if err := os.Remove(zipFileName); err != nil {
		log.Error("Unable to remove zip", "path", zipFileName, "err", err)
		os.Exit(1)
	}

	// Final steps
	for _, path := range flag.Args() {
		// Staple
		output, err := exec.Command("xcrun", "stapler", "staple", path).CombinedOutput()
		if err != nil {
			log.Error("Stapling failed", "path", path, "err", err, "output", string(output))
			os.Exit(1)
		}

		// Verify notarization
		output, err = exec.Command("codesign", "--test-requirement=\"=notarized\"", "--verify", path).CombinedOutput()
		if err != nil {
			log.Error("Final verification failed", "path", path, "err", err, "output", string(output))
			os.Exit(1)
		}
	}
}
