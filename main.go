package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

// Define variable
var isError bool = false

// Config ...
type Config struct {
	GMCloudSaaSEmail    string          `env:"email,required"`
	GMCloudSaaSPassword stepconf.Secret `env:"password,required"`
}

// install gmsaas if not installed.
func ensureGMSAASisInstalled() error {
	path, err := exec.LookPath("gmsaas")
	if err != nil {
		log.Infof("Installing gmsaas ...")
		cmd := command.New("pip3", "install", "gmsaas")
		if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
			return fmt.Errorf("%s failed, error: %s | output: %s", cmd.PrintableCommandArgs(), err, out)
		}
		log.Infof("gmsaas has been installed.")
	} else {
		log.Infof("gmsaas is already installed : %s", path)
	}

	// Set Custom user agent to improve customer support
	os.Setenv("GMSAAS_USER_AGENT_EXTRA_DATA", "bitrise.io")
	return nil
}

// printError prints an error.
func printError(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// abortf prints an error and terminates step
func abortf(format string, args ...interface{}) {
	printError(format, args...)
	os.Exit(1)
}

// setOperationFailed marked step as failed
func setOperationFailed(format string, args ...interface{}) {
	printError(format, args...)
	isError = true
}

func configureAndroidSDKPath() {
	log.Infof("Configure Android SDK configuration")

	value, exists := os.LookupEnv("ANDROID_HOME")
	if exists {
		cmd := command.New("gmsaas", "config", "set", "android-sdk-path", value)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			setOperationFailed("Failed to set android-sdk-path, error: error: %s | output: %s", cmd.PrintableCommandArgs(), err, out)
			return
		}
		log.Infof("Android SDK is configured")
	} else {
		setOperationFailed("Please set ANDROID_HOME environment variable")
		return
	}
}

func login(username, password string) {
	log.Infof("Login Genymotion Account")
	cmd := command.New("gmsaas", "auth", "login", username, password)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		abortf("Failed to log with gmsaas, error: error: %s | output: %s", cmd.PrintableCommandArgs(), err, out)
	}
	log.Infof("Logged to Genymotion Cloud SaaS platform")
}

func main() {

	var c Config
	if err := stepconf.Parse(&c); err != nil {
		abortf("Issue with input: %s", err)
	}
	stepconf.Print(c)

	if err := ensureGMSAASisInstalled(); err != nil {
		abortf("%s", err)
	}
	configureAndroidSDKPath()

	if err := tools.ExportEnvironmentWithEnvman("GMSAAS_USER_AGENT_EXTRA_DATA", "bitrise.io"); err != nil {
		printError("Failed to export %s, error: %v", "GMSAAS_USER_AGENT_EXTRA_DATA", err)
	}

	login(c.GMCloudSaaSEmail, string(c.GMCloudSaaSPassword))

	// --- Exit codes:
	// The exit code of your Step is very important. If you return
	//  with a 0 exit code `bitrise` will register your Step as "successful".
	// Any non zero exit code will be registered as "failed" by `bitrise`.
	if isError {
		// If at least one error happens, step will fail
		os.Exit(1)
	}
	os.Exit(0)
}
